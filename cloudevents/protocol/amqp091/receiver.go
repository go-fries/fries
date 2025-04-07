package amqp091

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/cloudevents/sdk-go/v2/binding"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Receiver struct {
	channel     *amqp.Channel
	queue       string
	deliveries  <-chan amqp.Delivery
	consumeOnce sync.Once
	consumer    string
}

func NewReceiver(channel *amqp.Channel, queue string, opts ...ReceiverOption) (*Receiver, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel cannot be nil")
	}

	r := &Receiver{
		channel: channel,
		queue:   queue,
	}

	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}

	deliveries, err := channel.Consume(
		queue,
		r.consumer,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	r.deliveries = deliveries
	return r, nil
}

func (r *Receiver) Receive(ctx context.Context) (binding.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case delivery, ok := <-r.deliveries:
		if !ok {
			return nil, io.EOF
		}
		return newMessage(&delivery), nil
	}
}

type ReceiverOption func(*Receiver) error

func WithConsumer(consumer string) ReceiverOption {
	return func(r *Receiver) error {
		r.consumer = consumer
		return nil
	}
}
