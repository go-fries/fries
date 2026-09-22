package queue

import (
	"context"

	"github.com/go-fries/fries/codec/v4"
)

// Definition binds a stable task name, payload type and codec without holding a
// producer or handler. It can be shared by producer and consumer code.
// Its zero value has an empty name: enqueueing returns ErrInvalidTaskType and
// handler registration is ignored.
type Definition[T any] struct {
	taskType string
	codec    codec.Codec
}

// Define creates a task definition with an explicit wire name. The name is used
// unchanged; it is not derived from T. Empty names behave like the zero definition.
// JSON is used unless WithCodec selects another encoding for both enqueueing
// and handling. Producers and consumers must use compatible definition options.
func Define[T any](taskType string, opts ...DefinitionOption) Definition[T] {
	c := newDefinitionConfig(opts...)
	return Definition[T]{taskType: taskType, codec: c.codec}
}

// TaskType returns the task name used in the durable task envelope.
func (d Definition[T]) TaskType() string {
	return d.taskType
}

// Enqueue encodes payload with the definition's codec and enqueues it with
// producer. Options, observer events and errors follow EnqueueForWithCodec.
// An empty name returns ErrInvalidTaskType before encoding or invoking the producer.
func (d Definition[T]) Enqueue(ctx context.Context, producer *Producer, payload T, opts ...EnqueueOption) (*Task, error) {
	if d.taskType == "" {
		return nil, ErrInvalidTaskType
	}
	return EnqueueForWithCodec(ctx, producer, d.taskType, payload, d.codec, opts...)
}

// Handle registers a payload-only function using the definition's codec.
// It delegates to HandlePayloadWithCodec: an empty name or nil function is ignored,
// and the last registration for a name wins. Decoding, middleware, retry and
// settlement behavior follow the existing Worker pipeline.
func (d Definition[T]) Handle(handler func(context.Context, T) error) WorkerOption {
	return HandlePayloadWithCodec(d.taskType, d.codec, handler)
}

// HandleFor registers a typed handler with access to the full TaskFor envelope,
// including task ID, attempt and metadata. It delegates to the package-level
// HandleForWithCodec with the definition's name and codec.
func (d Definition[T]) HandleFor(handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(d.taskType, d.codec, handler)
}
