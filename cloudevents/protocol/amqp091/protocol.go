package amqp091

import (
	"context"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Protocol struct {
	sender   *Sender
	receiver *Receiver
}

var _ protocol.Sender = (*Protocol)(nil)
var _ protocol.Receiver = (*Protocol)(nil)

func NewProtocol(sender *Sender, receiver *Receiver) *Protocol {
	return &Protocol{
		sender:   sender,
		receiver: receiver,
	}
}

func (p *Protocol) Send(ctx context.Context, in binding.Message, transformers ...binding.Transformer) error {
	return p.sender.Send(ctx, in, transformers...)
}

func (p *Protocol) Receive(ctx context.Context) (binding.Message, error) {
	return p.receiver.Receive(ctx)
}

type Config struct {
	Channel         *amqp.Channel
	Exchange        string
	RoutingKey      string
	Queue           string
	SenderOptions   []SenderOption
	ReceiverOptions []ReceiverOption
}

func NewProtocolFromConfig(config *Config) (*Protocol, error) {
	if err := config.Channel.ExchangeDeclare(
		config.Exchange, // name
		"topic",         // type
		true,            // durable
		false,           // auto-delete
		false,           // internal
		false,           // no-wait
		nil,             // args
	); err != nil {
		return nil, err
	}

	// if the queue doesn't exist, create it
	if _, err := config.Channel.QueueDeclare(
		config.Queue, // name
		true,         // durable
		false,        // auto-delete
		false,        // exclusive
		false,        // no-wait
		nil,          // args
	); err != nil {
		return nil, err
	}

	if err := config.Channel.QueueBind(
		config.Queue,      // queue name
		config.RoutingKey, // routing key
		config.Exchange,   // exchange
		false,
		nil,
	); err != nil {
		return nil, err
	}

	sender, err := NewSender(config.Channel, config.Exchange, config.RoutingKey, config.SenderOptions...)
	if err != nil {
		return nil, err
	}
	receiver, err := NewReceiver(config.Channel, config.Queue, config.ReceiverOptions...)
	if err != nil {
		return nil, err
	}
	return NewProtocol(sender, receiver), nil
}
