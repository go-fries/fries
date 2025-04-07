package amqp091

import (
	"bytes"
	"context"
	"strings"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/binding/format"
	"github.com/cloudevents/sdk-go/v2/binding/spec"
	amqp "github.com/rabbitmq/amqp091-go"
)

const prefix = "ce-"

type Message struct {
	delivery *amqp.Delivery
	version  spec.Version
	format   format.Format
}

func NewMessage(delivery *amqp.Delivery) *Message {
	return &Message{
		delivery: delivery,
		version:  spec.VS.Version("1.0"),
		format:   format.Lookup(delivery.ContentType),
	}
}

var (
	_ binding.Message               = (*Message)(nil)
	_ binding.MessageMetadataReader = (*Message)(nil)
)

func (m *Message) ReadEncoding() binding.Encoding {
	if m.format != nil {
		return binding.EncodingStructured
	}
	return binding.EncodingBinary
}

func (m *Message) ReadStructured(ctx context.Context, encoder binding.StructuredWriter) error {
	if m.format == nil {
		return binding.ErrNotStructured
	}
	return encoder.SetStructuredEvent(ctx, m.format, bytes.NewReader(m.delivery.Body))
}

func (m *Message) ReadBinary(ctx context.Context, encoder binding.BinaryWriter) error {
	if m.format != nil {
		return binding.ErrNotBinary
	}

	for k, v := range m.delivery.Headers {
		if strings.HasPrefix(k, prefix) {
			attribute := m.version.Attribute(strings.TrimPrefix(k, prefix))
			if attribute != nil {
				if err := encoder.SetAttribute(attribute, v); err != nil {
					return err
				}
			} else {
				if err := encoder.SetExtension(strings.ToLower(strings.TrimPrefix(k, prefix)), v); err != nil {
					return err
				}
			}
		}
	}

	return encoder.SetData(bytes.NewReader(m.delivery.Body))
}

func (m *Message) GetAttribute(attributeKind spec.Kind) (spec.Attribute, interface{}) {
	attribute := m.version.AttributeFromKind(attributeKind)
	if attribute != nil {
		return attribute, m.delivery.Headers[attribute.PrefixedName()]
	}
	return nil, nil
}

func (m *Message) GetExtension(name string) interface{} {
	return m.delivery.Headers[prefix+strings.ToLower(name)]
}

func (m *Message) Finish(err error) error {
	if err != nil {
		return m.delivery.Reject(false)
	}
	return m.delivery.Ack(false)
}
