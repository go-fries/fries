package queue

import "context"

// Queue stores tasks and creates consumers for task delivery.
type Queue interface {
	// Enqueue stores a task for future delivery.
	Enqueue(ctx context.Context, task *Task) error
	// NewConsumer creates a consumer for queue.
	NewConsumer(ctx context.Context, queue string) (Consumer, error)
}
