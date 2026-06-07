package queue

import (
	"context"
	"encoding/json"

	"github.com/go-fries/fries/codec/v3"
)

// JSONPayloadCodec is the default JSON codec used by typed payload helpers.
var JSONPayloadCodec codec.Codec = jsonPayloadCodec{}

type jsonPayloadCodec struct{}

func (jsonPayloadCodec) Marshal(data any) ([]byte, error) {
	return json.Marshal(data)
}

func (jsonPayloadCodec) Unmarshal(src []byte, dest any) error {
	return json.Unmarshal(src, dest)
}

// TaskFor is a typed view of a task with its payload decoded as T.
type TaskFor[T any] struct {
	// Task is the original queue task with its raw payload and delivery metadata.
	Task *Task

	// Payload is the decoded application payload.
	Payload T
}

// HandlerFor processes a task whose payload has been decoded as T.
type HandlerFor[T any] interface {
	// Handle processes a typed task and returns nil only when it should be acknowledged.
	Handle(ctx context.Context, task *TaskFor[T]) error
}

// HandlerFuncFor adapts a function to HandlerFor.
type HandlerFuncFor[T any] func(ctx context.Context, task *TaskFor[T]) error

// Handle calls f(ctx, task).
func (f HandlerFuncFor[T]) Handle(ctx context.Context, task *TaskFor[T]) error {
	return f(ctx, task)
}

// EnqueueFor encodes payload with JSONPayloadCodec and enqueues it as taskType.
func EnqueueFor[T any](ctx context.Context, producer *Producer, taskType string, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueForWithCodec(ctx, producer, taskType, payload, JSONPayloadCodec, opts...)
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
		codec = JSONPayloadCodec
	}

	data, err := codec.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return producer.Enqueue(ctx, taskType, data, opts...)
}

// HandleFor decodes task payloads with JSONPayloadCodec before calling handler.
func HandleFor[T any](taskType string, handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(taskType, JSONPayloadCodec, handler)
}

// HandleForWithCodec decodes task payloads with codec before calling handler.
func HandleForWithCodec[T any](taskType string, codec codec.Codec, handler HandlerFor[T]) WorkerOption {
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
