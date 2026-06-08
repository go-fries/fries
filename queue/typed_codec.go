package queue

import (
	"context"
	"encoding/json"

	"github.com/go-fries/fries/codec/v3"
)

var defaultCodec codec.Codec = jsonCodec{}

type jsonCodec struct{}

func (jsonCodec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (jsonCodec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}

// EnqueueFor encodes payload with the default JSON codec and enqueues it as taskType.
func EnqueueFor[T any](ctx context.Context, producer *Producer, taskType string, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueForWithCodec(ctx, producer, taskType, payload, defaultCodec, opts...)
}

// EnqueueForWithCodec encodes payload with codec and enqueues it as taskType.
func EnqueueForWithCodec[T any](
	ctx context.Context,
	producer *Producer,
	taskType string,
	payload T,
	codec codec.Codec,
	opts ...EnqueueOption,
) (*Task, error) {
	if codec == nil {
		codec = defaultCodec
	}

	data, err := codec.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return producer.Enqueue(ctx, taskType, data, opts...)
}

// HandleFor decodes task payloads with the default JSON codec before calling handler.
func HandleFor[T any](taskType string, handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(taskType, defaultCodec, handler)
}

// HandleForWithCodec decodes task payloads with codec before calling handler.
func HandleForWithCodec[T any](taskType string, codec codec.Codec, handler HandlerFor[T]) WorkerOption {
	if handler == nil {
		return Handle(taskType, nil)
	}
	if codec == nil {
		codec = defaultCodec
	}

	return Handle(taskType, HandlerFunc(func(ctx context.Context, task *Task) error {
		var payload T
		if err := codec.Unmarshal(task.Payload, &payload); err != nil {
			return err
		}
		return handler.Handle(ctx, &TaskFor[T]{
			Task:    task,
			Payload: payload,
		})
	}))
}
