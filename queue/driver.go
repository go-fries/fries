package queue

import "context"

type Driver interface {
	Enqueue(ctx context.Context, queue string, data []byte) error
	Dequeue(ctx context.Context, queue string) ([]byte, error)
}
