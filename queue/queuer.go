package queue

import (
	"context"
)

type Queuer interface {
	// Enqueue adds a message to the queue.
	Enqueue(ctx context.Context, queue string, data []byte) error

	// Dequeue retrieves a message from the queue.
	Dequeue(ctx context.Context, queue string) ([]byte, error)

	// Len returns the number of messages in the queue.
	Len(ctx context.Context, queue string) (int64, error)

	// IsEmpty checks if the queue is empty.
	IsEmpty(ctx context.Context, queue string) (bool, error)

	// Peek retrieves the next message without removing it from the queue.
	Peek(ctx context.Context, queue string) ([]byte, error)

	// Drain removes all messages from the queue.
	Drain(ctx context.Context, queue string) error
}
