package channel

import (
	"context"
	"sync"

	"github.com/go-fries/fries/queue/v3"
)

type Queuer struct {
	data map[string]chan []byte
	mu   sync.Mutex
	size int64 // the capacity of the channel
}

type Option func(*Queuer)

// WithSize sets the size of the channel.
func WithSize(size int64) Option {
	return func(q *Queuer) {
		q.size = size
	}
}

var _ queue.Queuer = (*Queuer)(nil)

func NewQueuer() *Queuer {
	return &Queuer{
		data: make(map[string]chan []byte),
		size: 100,
	}
}

func (q *Queuer) Enqueue(ctx context.Context, queue string, data []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.data[queue]; !exists {
		q.data[queue] = make(chan []byte, q.size) // Create a buffered channel with a capacity of 100
	}

	select {
	case q.data[queue] <- data:
		return nil
	case <-ctx.Done():
		return ctx.Err() // Return context error if the context is done
	}
}

func (q *Queuer) Dequeue(ctx context.Context, queue string) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch, exists := q.data[queue]
	if !exists {
		return nil, nil // Return nil if the queue does not exist
	}

	select {
	case data := <-ch:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err() // Return context error if the context is done
	}
}

func (q *Queuer) Len(ctx context.Context, queue string) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch, exists := q.data[queue]
	if !exists {
		return 0, nil // Return 0 if the queue does not exist
	}

	return int64(len(ch)), nil // Return the length of the channel
}

func (q *Queuer) IsEmpty(ctx context.Context, queue string) (bool, error) {
	length, err := q.Len(ctx, queue)
	if err != nil {
		return false, err
	}
	return length == 0, nil
}

func (q *Queuer) Peek(ctx context.Context, queue string) ([]byte, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch, exists := q.data[queue]
	if !exists || len(ch) == 0 {
		return nil, nil // Return nil if the queue does not exist or is empty
	}

	select {
	case data := <-ch: // Peek the first element
		ch <- data // Put it back into the channel
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err() // Return context error if the context is done
	}
}

func (q *Queuer) Drain(ctx context.Context, queue string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, exists := q.data[queue]; !exists {
		return nil
	}

	close(q.data[queue])  // Close the channel to release resources
	delete(q.data, queue) // Remove the queue from the map
	return nil
}
