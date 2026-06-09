package rabbitmq

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

const defaultExchange = ""

var (
	errNilConnection        = errors.New("queue/adapter/rabbitmq: connection is nil")
	errNilChannelOpener     = errors.New("queue/adapter/rabbitmq: channel opener is nil")
	errNilChannel           = errors.New("queue/adapter/rabbitmq: channel is nil")
	errPublishNacked        = errors.New("queue/adapter/rabbitmq: publish not acknowledged")
	errPublishConfirmClosed = errors.New("queue/adapter/rabbitmq: publish confirmation channel closed")
)

type channel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	Confirm(noWait bool) error
	NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Close() error
}

type channelOpener func(ctx context.Context) (channel, error)

// Queue stores and consumes queue tasks with RabbitMQ.
type Queue struct {
	opener           channelOpener
	prefix           string
	durable          bool
	delayQueueTTL    time.Duration
	prefetch         int
	publisherConfirm bool
}

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates a RabbitMQ queue adapter using connection.
func NewQueue(connection *amqp.Connection, opts ...Option) *Queue {
	return newQueue(func(ctx context.Context) (channel, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if connection == nil {
			return nil, errNilConnection
		}
		return connection.Channel()
	}, opts...)
}

func newQueue(opener channelOpener, opts ...Option) *Queue {
	c := newConfig(opts...)
	return &Queue{
		opener:           opener,
		prefix:           c.prefix,
		durable:          c.durable,
		delayQueueTTL:    c.delayQueueTTL,
		prefetch:         c.prefetch,
		publisherConfirm: c.publisherConfirm,
	}
}

// Enqueue stores task in a ready queue or delay queue.
func (q *Queue) Enqueue(ctx context.Context, task *queue.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task = task.Clone()
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}
	return q.publishTask(ctx, task, delayFromAvailableAt(time.Now().UTC(), task.AvailableAt))
}

// NewConsumer creates a RabbitMQ consumer using config.
func (q *Queue) NewConsumer(ctx context.Context, config queue.ConsumerConfig) (queue.Consumer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config = config.Normalize()

	ch, err := q.openChannel(ctx)
	if err != nil {
		return nil, err
	}
	if err := q.ensureReadyQueue(ch, config.Queue); err != nil {
		_ = ch.Close()
		return nil, err
	}
	if err := ch.Qos(q.prefetch, 0, false); err != nil {
		_ = ch.Close()
		return nil, err
	}

	deliveries, err := ch.Consume(q.readyQueueName(config.Queue), config.Name, false, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		return nil, err
	}
	return &consumer{
		queue:      q,
		channel:    ch,
		deliveries: deliveries,
	}, nil
}

func (q *Queue) publishTask(ctx context.Context, task *queue.Task, delay time.Duration) error {
	msg, err := publishingFromTask(task)
	if err != nil {
		return err
	}

	return q.withChannel(ctx, func(ch channel) error {
		if err := q.ensureReadyQueue(ch, task.Queue); err != nil {
			return err
		}

		target := q.readyQueueName(task.Queue)
		if delay > 0 {
			target = q.delayQueueName(task.Queue, delay)
			if err := q.ensureDelayQueue(ch, task.Queue, target, delay); err != nil {
				return err
			}
		}
		return q.publish(ctx, ch, defaultExchange, target, msg)
	})
}

func (q *Queue) publish(ctx context.Context, ch channel, exchange, key string, msg amqp.Publishing) error {
	if !q.publisherConfirm {
		return ch.PublishWithContext(ctx, exchange, key, false, false, msg)
	}

	if err := ch.Confirm(false); err != nil {
		return err
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	if err := ch.PublishWithContext(ctx, exchange, key, false, false, msg); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case confirmation, ok := <-confirms:
		if !ok {
			return errPublishConfirmClosed
		}
		if !confirmation.Ack {
			return errPublishNacked
		}
		return nil
	}
}

func (q *Queue) ensureReadyQueue(ch channel, name string) error {
	queueName := q.readyQueueName(name)
	_, err := ch.QueueDeclare(queueName, q.durable, false, false, false, nil)
	return queueDeclareError(queueName, err)
}

func (q *Queue) ensureDeadLetterQueue(ch channel, name string) error {
	queueName := q.deadLetterQueueName(name)
	_, err := ch.QueueDeclare(queueName, q.durable, false, false, false, nil)
	return queueDeclareError(queueName, err)
}

func (q *Queue) ensureDelayQueue(ch channel, queueName, delayQueueName string, delay time.Duration) error {
	expires := delay + q.delayQueueTTL
	_, err := ch.QueueDeclare(delayQueueName, q.durable, false, false, false, amqp.Table{
		"x-message-ttl":             durationMillis(delay),
		"x-expires":                 durationMillis(expires),
		"x-dead-letter-exchange":    defaultExchange,
		"x-dead-letter-routing-key": q.readyQueueName(queueName),
	})
	return queueDeclareError(delayQueueName, err)
}

func (q *Queue) openChannel(ctx context.Context) (channel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if q.opener == nil {
		return nil, errNilChannelOpener
	}
	ch, err := q.opener(ctx)
	if err != nil {
		return nil, err
	}
	if ch == nil {
		return nil, errNilChannel
	}
	return ch, nil
}

func (q *Queue) withChannel(ctx context.Context, fn func(channel) error) error {
	ch, err := q.openChannel(ctx)
	if err != nil {
		return err
	}
	err = fn(ch)
	closeErr := ch.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func (q *Queue) readyQueueName(name string) string {
	return q.queueName(name)
}

func (q *Queue) delayQueueName(name string, delay time.Duration) string {
	return q.queueName(name) + ".delay." + strconv.FormatInt(durationMillis(delay), 10)
}

func (q *Queue) deadLetterQueueName(name string) string {
	return q.queueName(name) + ".dead"
}

func (q *Queue) queueName(name string) string {
	if name == "" {
		name = queue.DefaultQueue
	}
	name = strings.TrimPrefix(name, ".")
	if q.prefix == "" {
		return name
	}
	return q.prefix + "." + name
}
