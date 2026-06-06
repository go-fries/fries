package queue

import (
	"context"
	"time"
)

type Backend interface {
	Enqueue(ctx context.Context, task *Task) error
	Dequeue(ctx context.Context, queue string, visibilityTimeout time.Duration) (*Lease, error)
	Ack(ctx context.Context, lease *Lease) error
	Retry(ctx context.Context, lease *Lease, delay time.Duration) error
	DeadLetter(ctx context.Context, lease *Lease, reason string) error
}
