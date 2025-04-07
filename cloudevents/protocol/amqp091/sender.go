package amqp091

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Sender implements the CloudEvents sender for AMQP 0.9.1 protocol
type Sender struct {
	channel    *amqp.Channel
	exchange   string
	routingKey string
	mandatory  bool
	immediate  bool
	persistent bool
}

var _ protocol.Sender = (*Sender)(nil)

func NewSender(channel *amqp.Channel, exchange, routingKey string, opts ...SenderOption) (*Sender, error) {
	s := &Sender{
		channel:    channel,
		exchange:   exchange,
		routingKey: routingKey,
		persistent: true, // 默认使用持久化消息
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// Send implements the CloudEvents sender interface
func (s *Sender) Send(ctx context.Context, in binding.Message, transformers ...binding.Transformer) (err error) {
	defer func() {
		_ = in.Finish(err)
	}()

	msg := &amqp.Publishing{
		DeliveryMode: amqp.Persistent,
	}

	if !s.persistent {
		msg.DeliveryMode = amqp.Transient
	}

	err = writeMessage(ctx, in, msg, transformers...)
	if err != nil {
		return err
	}

	err = s.channel.PublishWithContext(
		ctx,
		s.exchange,
		s.routingKey,
		s.mandatory,
		s.immediate,
		*msg,
	)
	return err
}

// SenderOption is a function that can modify a sender
type SenderOption func(*Sender) error

// WithMandatory sets the mandatory flag for message publishing
func WithMandatory(mandatory bool) SenderOption {
	return func(s *Sender) error {
		s.mandatory = mandatory
		return nil
	}
}

// WithImmediate sets the immediate flag for message publishing
func WithImmediate(immediate bool) SenderOption {
	return func(s *Sender) error {
		s.immediate = immediate
		return nil
	}
}

// WithPersistent sets the persistent flag for message publishing
func WithPersistent(persistent bool) SenderOption {
	return func(s *Sender) error {
		s.persistent = persistent
		return nil
	}
}
