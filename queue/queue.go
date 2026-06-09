package queue

import (
	"context"
	"time"
)

// Queue stores tasks and creates consumers for task delivery.
type Queue interface {
	// Enqueue stores a task for future delivery.
	Enqueue(ctx context.Context, task *Task) error
	// NewConsumer creates a consumer for queue.
	NewConsumer(ctx context.Context, queue string) (Consumer, error)
}

// Consumer receives task deliveries from a queue.
type Consumer interface {
	// Receive blocks until a delivery is available or ctx is canceled.
	Receive(ctx context.Context) (Delivery, error)
	// Close stops the consumer and releases backend resources.
	Close() error
}

// Delivery represents one delivery attempt of a task.
//
// Queue implementations may attach backend-specific acknowledgement state to a
// delivery. That delivery state is intentionally separate from Task.Metadata.
type Delivery interface {
	// Task returns the delivered task envelope.
	Task() *Task
	// Ack marks the delivery as successfully processed.
	Ack(ctx context.Context) error
	// Retry releases the delivery for another attempt after delay.
	Retry(ctx context.Context, delay time.Duration) error
	// DeadLetter moves the delivery out of normal processing with a reason.
	DeadLetter(ctx context.Context, reason string) error
}
