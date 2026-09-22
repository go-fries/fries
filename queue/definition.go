package queue

import (
	"context"

	"github.com/go-fries/fries/codec/v4"
)

// Definition binds a stable task name to a payload type without holding a
// producer, handler or codec. It can be shared by producer and consumer code.
// Its zero value has an empty name: enqueueing returns ErrInvalidTaskType and
// handler registration is ignored.
type Definition[T any] struct {
	taskType string
}

// Define creates a task definition with an explicit wire name. The name is used
// unchanged; it is not derived from T. Empty names behave like the zero definition.
func Define[T any](taskType string) Definition[T] {
	return Definition[T]{taskType: taskType}
}

// TaskType returns the task name used in the durable task envelope.
func (d Definition[T]) TaskType() string {
	return d.taskType
}

// Enqueue encodes payload as JSON and enqueues it with producer. All enqueue
// options, observer events and backend errors follow EnqueueFor. An empty name
// returns ErrInvalidTaskType before encoding or invoking the producer.
func (d Definition[T]) Enqueue(ctx context.Context, producer *Producer, payload T, opts ...EnqueueOption) (*Task, error) {
	return d.EnqueueWithCodec(ctx, producer, payload, nil, opts...)
}

// EnqueueWithCodec is Enqueue with a custom codec. A nil codec selects JSON.
// Configure the consumer with a matching codec using HandleWithCodec or
// HandleForWithCodec; codecs are not stored in the definition or task envelope.
func (d Definition[T]) EnqueueWithCodec(ctx context.Context, producer *Producer, payload T, codec codec.Codec, opts ...EnqueueOption) (*Task, error) {
	if d.taskType == "" {
		return nil, ErrInvalidTaskType
	}
	return EnqueueForWithCodec(ctx, producer, d.taskType, payload, codec, opts...)
}

// Handle registers a payload-only function using the default JSON codec.
// It delegates to HandlePayload: an empty name or nil function is ignored,
// and the last registration for a name wins. Decoding, middleware, retry and
// settlement behavior follow the existing Worker pipeline.
func (d Definition[T]) Handle(handler func(context.Context, T) error) WorkerOption {
	return HandlePayload(d.taskType, handler)
}

// HandleWithCodec is Handle with a custom codec. A nil codec selects JSON.
func (d Definition[T]) HandleWithCodec(codec codec.Codec, handler func(context.Context, T) error) WorkerOption {
	return HandlePayloadWithCodec(d.taskType, codec, handler)
}

// HandleFor registers a typed handler with access to the full TaskFor envelope,
// including task ID, attempt and metadata. It delegates to the package-level
// HandleFor with the definition's name and the default JSON codec.
func (d Definition[T]) HandleFor(handler HandlerFor[T]) WorkerOption {
	return HandleFor(d.taskType, handler)
}

// HandleForWithCodec is HandleFor with a custom codec. A nil codec selects JSON.
func (d Definition[T]) HandleForWithCodec(codec codec.Codec, handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(d.taskType, codec, handler)
}
