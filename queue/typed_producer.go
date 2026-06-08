package queue

import "context"

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
