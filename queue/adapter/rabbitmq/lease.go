package rabbitmq

import (
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/queue/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consumer struct {
	queue      *Queue
	channel    channel
	deliveries <-chan amqp.Delivery
	closeOnce  sync.Once
	closeErr   error
}

func (c *consumer) Receive(ctx context.Context) (queue.Delivery, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case delivery, ok := <-c.deliveries:
		if !ok {
			return nil, queue.ErrConsumerClosed
		}
		leased, err := deliveryFromAMQP(delivery)
		if err != nil {
			_ = delivery.Reject(false)
			return nil, err
		}
		leased.queue = c.queue
		return &leased, nil
	}
}

func (c *consumer) Close() error {
	if c == nil || c.channel == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		c.closeErr = c.channel.Close()
	})
	return c.closeErr
}

type delivery struct {
	queue    *Queue
	task     *queue.Task
	delivery amqp.Delivery
}

func (d *delivery) Task() *queue.Task {
	if d == nil {
		return nil
	}
	return d.task
}

func (d *delivery) Ack(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}
	return d.delivery.Ack(false)
}

func (d *delivery) Retry(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	if err := d.queue.publishTask(ctx, task, delay); err != nil {
		return err
	}
	return d.Ack(ctx)
}

func (d *delivery) DeadLetter(ctx context.Context, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.Clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata[deadReasonKey] = reason

	msg, err := publishingFromTask(task)
	if err != nil {
		return err
	}
	msg.Headers = amqp.Table{deadReasonKey: reason}
	err = d.queue.withChannel(ctx, func(ch channel) error {
		if err := d.queue.ensureDeadLetterQueue(ch, task.Queue); err != nil {
			return err
		}
		return d.queue.publish(ctx, ch, defaultExchange, d.queue.deadLetterQueueName(task.Queue), msg)
	})
	if err != nil {
		return err
	}
	return d.Ack(ctx)
}
