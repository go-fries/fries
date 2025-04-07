package amqp091

import (
	"context"
	"io"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/binding/format"
	"github.com/cloudevents/sdk-go/v2/binding/spec"
	"github.com/cloudevents/sdk-go/v2/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

func writeMessage(ctx context.Context, in binding.Message, amqpMessage *amqp.Publishing, transformers ...binding.Transformer) error {
	structuredWriter := newAMQPMessageWriter(amqpMessage)
	binaryWriter := newAMQPMessageWriter(amqpMessage)

	_, err := binding.Write(
		ctx,
		in,
		structuredWriter,
		binaryWriter,
		transformers...,
	)
	return err
}

type amqpMessageWriter struct {
	message *amqp.Publishing
	version spec.Version
}

var (
	_ binding.BinaryWriter     = (*amqpMessageWriter)(nil)
	_ binding.StructuredWriter = (*amqpMessageWriter)(nil)
)

func newAMQPMessageWriter(message *amqp.Publishing) *amqpMessageWriter {
	return &amqpMessageWriter{
		message: message,
		version: spec.VS.Version("1.0"),
	}
}

func (a *amqpMessageWriter) SetAttribute(attribute spec.Attribute, value any) error {
	if attribute.Kind() == spec.DataContentType {
		if value == nil {
			a.message.ContentType = ""
			return nil
		}
		s, err := types.Format(value)
		if err != nil {
			return err
		}
		a.message.ContentType = s
		return nil
	}

	if value == nil {
		delete(a.message.Headers, prefix+attribute.Name())
		return nil
	}

	v, err := safeAMQPHeadersUnwrap(value)
	if err != nil {
		return err
	}
	a.message.Headers[prefix+attribute.Name()] = v

	return nil
}

func (a *amqpMessageWriter) SetExtension(name string, value any) error {
	if value == nil {
		return nil
	}

	if a.message.Headers == nil {
		a.message.Headers = make(amqp.Table)
	}

	val, err := safeAMQPHeadersUnwrap(value)
	if err != nil {
		return err
	}

	a.message.Headers[prefix+name] = val
	return nil
}

func (a *amqpMessageWriter) SetData(data io.Reader) error {
	bytes, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	a.message.Body = bytes
	return nil
}

func (a *amqpMessageWriter) SetStructuredEvent(ctx context.Context, fmt format.Format, event io.Reader) error {
	if event == nil {
		return nil
	}

	bytes, err := io.ReadAll(event)
	if err != nil {
		return err
	}

	a.message.ContentType = fmt.MediaType()
	a.message.Body = bytes
	return nil
}

func (a *amqpMessageWriter) Start(ctx context.Context) error {
	if a.message.Headers == nil {
		a.message.Headers = make(amqp.Table)
	}
	return nil
}

func (a *amqpMessageWriter) End(context.Context) error {
	return nil
}

func safeAMQPHeadersUnwrap(value interface{}) (interface{}, error) {
	v, err := types.Validate(value)
	if err != nil {
		return nil, err
	}

	switch tv := v.(type) {
	case types.URI:
		v = tv.String()
	case types.URIRef:
		v = tv.String()
	case types.Timestamp:
		v = tv.Time
	case int32:
		v = int64(tv)
	}
	return v, nil
}
