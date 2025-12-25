package queue

import "context"

// Driver defines the queue driver interface (implemented by channel/redis/amqp etc.)
type Driver interface {
	// Push pushes a job to the queue (immediately available or delayed based on job.AvailableAt)
	Push(ctx context.Context, job Job) error

	// Pop retrieves a job from the queue
	// - Blocks until a job is available or context is cancelled
	// - Returns jobs sorted by priority (higher priority first)
	// - Only returns jobs where availableAt <= now
	Pop(ctx context.Context, queues ...string) (Job, error)

	// Ack acknowledges job completion, removes it from the queue
	Ack(ctx context.Context, job Job) error

	// Fail marks a job as failed
	// - If attempts < maxAttempts, re-queue with delay (retry)
	// - If attempts >= maxAttempts, move to dead letter queue
	Fail(ctx context.Context, job Job, err error) error

	// Size returns the number of pending jobs in the queue
	Size(ctx context.Context, queue string) (int64, error)

	// Dead returns the number of jobs in the dead letter queue
	Dead(ctx context.Context, queue string) (int64, error)

	// Clear clears all jobs from the specified queue
	Clear(ctx context.Context, queue string) error

	// ClearDead clears all jobs from the dead letter queue
	ClearDead(ctx context.Context, queue string) error
}

// Startable is an optional interface for drivers that need to start background tasks
type Startable interface {
	Start(ctx context.Context) error
}

// Stoppable is an optional interface for drivers that need graceful shutdown
type Stoppable interface {
	Stop(ctx context.Context) error
}

// Retryable is an optional interface for drivers that support retrying dead jobs
type Retryable interface {
	// Retry moves a job from the dead letter queue back to the main queue
	Retry(ctx context.Context, queue string, jobID string) error

	// RetryAll moves all jobs from the dead letter queue back to the main queue
	RetryAll(ctx context.Context, queue string) (int64, error)
}

// Inspectable is an optional interface for drivers that support queue inspection (for debugging)
type Inspectable interface {
	// Peek returns jobs from the queue without removing them
	Peek(ctx context.Context, queue string, limit int) ([]Job, error)

	// PeekDead returns jobs from the dead letter queue without removing them
	PeekDead(ctx context.Context, queue string, limit int) ([]Job, error)
}
