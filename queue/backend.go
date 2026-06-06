package queue

import (
	"context"
	"time"
)

// Backend stores tasks and manages task delivery state.
type Backend interface {
	// Enqueue stores a task for future delivery.
	Enqueue(ctx context.Context, task *Task) error
	// Dequeue returns an available task lease from the queue.
	Dequeue(ctx context.Context, queue string, visibilityTimeout time.Duration) (*Lease, error)
	// Ack marks a leased task as successfully processed.
	Ack(ctx context.Context, lease *Lease) error
	// Retry releases a leased task for another attempt after delay.
	Retry(ctx context.Context, lease *Lease, delay time.Duration) error
	// DeadLetter moves a leased task out of normal processing with a reason.
	DeadLetter(ctx context.Context, lease *Lease, reason string) error
}
