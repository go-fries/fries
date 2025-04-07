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
	Channal         *amqp.Channel
	Exchange        string
	RoutingKey      string
	Queue           string
	SenderOptions   []SenderOption
	ReceiverOptions []ReceiverOption
}

func NewProtocolFromConfig(config *Config) (*Protocol, error) {
	sender, err := NewSender(config.Channal, config.Exchange, config.RoutingKey, config.SenderOptions...)
	if err != nil {
		return nil, err
	}
	receiver, err := NewReceiver(config.Channal, config.Queue, config.ReceiverOptions...)
	if err != nil {
		return nil, err
	}
	return NewProtocol(sender, receiver), nil
}
