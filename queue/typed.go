package queue

import (
	"context"
	"encoding/json"
)

type PayloadCodec interface {
	Marshal(data any) ([]byte, error)
	Unmarshal(src []byte, dest any) error
}

var JSONPayloadCodec PayloadCodec = jsonPayloadCodec{}

type jsonPayloadCodec struct{}

func (jsonPayloadCodec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (jsonPayloadCodec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}

type TaskFor[T any] struct {
	*Task
	Payload T
}

type HandlerFor[T any] interface {
	Handle(ctx context.Context, task *TaskFor[T]) error
}

type HandlerFuncFor[T any] func(ctx context.Context, task *TaskFor[T]) error

func (f HandlerFuncFor[T]) Handle(ctx context.Context, task *TaskFor[T]) error {
	return f(ctx, task)
}

func EnqueueFor[T any](ctx context.Context, producer *Producer, taskType string, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueForWithCodec(ctx, producer, taskType, payload, JSONPayloadCodec, opts...)
}

func EnqueueForWithCodec[T any](
	ctx context.Context,
	producer *Producer,
	taskType string,
	payload T,
	codec PayloadCodec,
	opts ...EnqueueOption,
) (*Task, error) {
	if codec == nil {
		codec = JSONPayloadCodec
	}

	data, err := codec.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return producer.Enqueue(ctx, taskType, data, opts...)
}

func HandleFor[T any](taskType string, handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(taskType, JSONPayloadCodec, handler)
}

func HandleForWithCodec[T any](taskType string, codec PayloadCodec, handler HandlerFor[T]) WorkerOption {
	if handler == nil {
		return Handle(taskType, nil)
	}
	if codec == nil {
		codec = JSONPayloadCodec
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
