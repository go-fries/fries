package amqp091

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Receiver struct {
	channel  *amqp.Channel
	queue    string
	consumer string
	incoming chan amqp.Delivery
}

var (
	_ protocol.Receiver = (*Receiver)(nil)
	_ protocol.Opener   = (*Receiver)(nil)
)

func NewReceiver(channel *amqp.Channel, queue string, opts ...ReceiverOption) (*Receiver, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel cannot be nil")
	}

	r := &Receiver{
		channel:  channel,
		queue:    queue,
		incoming: make(chan amqp.Delivery),
	}

	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func (r *Receiver) Receive(ctx context.Context) (binding.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case delivery, ok := <-r.incoming:
		if !ok {
			return nil, io.EOF
		}
		return newMessage(&delivery), nil
	}
}

func (r *Receiver) OpenInbound(context.Context) error {
	deliveries, err := r.channel.Consume(
		r.queue,
		r.consumer,
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	go func() {
		defer close(r.incoming)
		for {
			select {
			case delivery, ok := <-deliveries:
				if !ok {
					return
				}
				r.incoming <- delivery
			}
		}
	}()

	return nil
}

type ReceiverOption func(*Receiver) error

func WithConsumer(consumer string) ReceiverOption {
	return func(r *Receiver) error {
		r.consumer = consumer
		return nil
	}
}
