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

// TaskProducer enqueues one task type with a structured payload.
type TaskProducer[T any] struct {
	producer *Producer
	taskType string
}

// NewTaskProducer creates a typed producer bound to taskType.
func NewTaskProducer[T any](producer *Producer, taskType string) *TaskProducer[T] {
	return &TaskProducer[T]{
		producer: producer,
		taskType: taskType,
	}
}

// Enqueue encodes payload with the default JSON codec and enqueues it as the producer task type.
func (p *TaskProducer[T]) Enqueue(ctx context.Context, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueFor(ctx, p.producer, p.taskType, payload, opts...)
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
