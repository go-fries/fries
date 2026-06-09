package queue

import (
	"context"
	"time"
)

// ConsumerConfig configures a consumer created by a Queue.
type ConsumerConfig struct {
	// Queue is the logical queue name to consume. Empty values are treated as
	// DefaultQueue.
	Queue string

	// Name identifies the consumer instance when a backend supports or requires
	// consumer identity. Empty values let the queue implementation choose its
	// default behavior.
	Name string
}

// Normalize returns a copy of c with shared queue defaults applied.
func (c ConsumerConfig) Normalize() ConsumerConfig {
	if c.Queue == "" {
		c.Queue = DefaultQueue
	}
	return c
}

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
