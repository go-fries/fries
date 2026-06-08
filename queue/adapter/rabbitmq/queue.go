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

var errNilChannel = errors.New("queue/adapter/rabbitmq: channel is nil")

// Channel is the subset of an AMQP 0.9.1 channel used by Queue.
type Channel interface {
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Get(queue string, autoAck bool) (amqp.Delivery, bool, error)
}

// Queue stores and consumes queue tasks with RabbitMQ.
type Queue struct {
	channel       Channel
	prefix        string
	durable       bool
	delayQueueTTL time.Duration
}

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates a RabbitMQ queue adapter using channel.
func NewQueue(channel Channel, opts ...Option) *Queue {
	c := newConfig(opts...)
	return &Queue{
		channel:       channel,
		prefix:        c.prefix,
		durable:       c.durable,
		delayQueueTTL: c.delayQueueTTL,
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

// Dequeue returns one available task lease from queueName.
func (q *Queue) Dequeue(ctx context.Context, queueName string, _ time.Duration) (queue.Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queueName == "" {
		queueName = queue.DefaultQueue
	}
	if err := q.ensureReadyQueue(queueName); err != nil {
		return nil, err
	}

	delivery, ok, err := q.channel.Get(q.readyQueueName(queueName), false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, queue.ErrNoTask
	}

	return leaseFromDelivery(delivery)
}

// Ack acknowledges a leased RabbitMQ delivery.
func (q *Queue) Ack(ctx context.Context, lease queue.Lease) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l, ok := lease.(*rabbitLease)
	if !ok || l == nil || l.task == nil {
		return nil
	}
	return l.delivery.Ack(false)
}

// Retry republishes a leased task after delay and acknowledges the original delivery.
func (q *Queue) Retry(ctx context.Context, lease queue.Lease, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := q.publishTask(ctx, task, delay); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}

// DeadLetter publishes a leased task to the dead-letter queue and acknowledges the original delivery.
func (q *Queue) DeadLetter(ctx context.Context, lease queue.Lease, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().Clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata[deadReasonKey] = reason

	if err := q.ensureDeadLetterQueue(task.Queue); err != nil {
		return err
	}
	msg, err := publishingFromTask(task)
	if err != nil {
		return err
	}
	msg.Headers = amqp.Table{deadReasonKey: reason}
	if err := q.channel.PublishWithContext(ctx, defaultExchange, q.deadLetterQueueName(task.Queue), false, false, msg); err != nil {
		return err
	}
	return q.Ack(ctx, lease)
}

func (q *Queue) publishTask(ctx context.Context, task *queue.Task, delay time.Duration) error {
	if q.channel == nil {
		return errNilChannel
	}
	if err := q.ensureReadyQueue(task.Queue); err != nil {
		return err
	}

	msg, err := publishingFromTask(task)
	if err != nil {
		return err
	}

	target := q.readyQueueName(task.Queue)
	if delay > 0 {
		target = q.delayQueueName(task.Queue, delay)
		if err := q.ensureDelayQueue(task.Queue, target, delay); err != nil {
			return err
		}
	}
	return q.channel.PublishWithContext(ctx, defaultExchange, target, false, false, msg)
}

func (q *Queue) ensureReadyQueue(name string) error {
	if q.channel == nil {
		return errNilChannel
	}
	queueName := q.readyQueueName(name)
	_, err := q.channel.QueueDeclare(queueName, q.durable, false, false, false, nil)
	return queueDeclareError(queueName, err)
}

func (q *Queue) ensureDeadLetterQueue(name string) error {
	if q.channel == nil {
		return errNilChannel
	}
	queueName := q.deadLetterQueueName(name)
	_, err := q.channel.QueueDeclare(queueName, q.durable, false, false, false, nil)
	return queueDeclareError(queueName, err)
}

func (q *Queue) ensureDelayQueue(queueName, delayQueueName string, delay time.Duration) error {
	expires := delay + q.delayQueueTTL
	_, err := q.channel.QueueDeclare(delayQueueName, q.durable, false, false, false, amqp.Table{
		"x-message-ttl":             durationMillis(delay),
		"x-expires":                 durationMillis(expires),
		"x-dead-letter-exchange":    defaultExchange,
		"x-dead-letter-routing-key": q.readyQueueName(queueName),
	})
	return queueDeclareError(delayQueueName, err)
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
