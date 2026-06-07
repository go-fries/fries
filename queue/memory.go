package queue

import (
	"context"
	"sync"
	"time"
)

// MemoryQueue is an in-memory Queue implementation for tests and local use.
type MemoryQueue struct {
	mu         sync.Mutex
	queues     map[string][]*Task
	deadLetter map[string][]*Task
}

var _ Queue = (*MemoryQueue)(nil)

// NewMemoryQueue creates an empty in-memory queue.
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{
		queues:     make(map[string][]*Task),
		deadLetter: make(map[string][]*Task),
	}
}

// Enqueue stores task in memory.
func (b *MemoryQueue) Enqueue(ctx context.Context, task *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	task = task.clone()
	if task.Queue == "" {
		task.Queue = DefaultQueue
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.queues[task.Queue] = append(b.queues[task.Queue], task)
	return nil
}

// Dequeue returns the first available task from queue.
func (b *MemoryQueue) Dequeue(ctx context.Context, queue string, _ time.Duration) (*Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queue == "" {
		queue = DefaultQueue
	}

	now := time.Now().UTC()
	b.mu.Lock()
	defer b.mu.Unlock()

	tasks := b.queues[queue]
	for i, task := range tasks {
		if task.AvailableAt.After(now) {
			continue
		}

		b.queues[queue] = append(tasks[:i], tasks[i+1:]...)
		task = task.clone()
		task.Attempt++
		return &Lease{Task: task, Token: task.ID}, nil
	}

	return nil, ErrNoTask
}

// Ack marks a memory lease as complete.
func (b *MemoryQueue) Ack(ctx context.Context, _ *Lease) error {
	return ctx.Err()
}

// Retry re-enqueues a leased task after delay.
func (b *MemoryQueue) Retry(ctx context.Context, lease *Lease, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task == nil {
		return nil
	}

	task := lease.Task.clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	return b.Enqueue(ctx, task)
}

// DeadLetter stores a leased task in the in-memory dead-letter list.
func (b *MemoryQueue) DeadLetter(ctx context.Context, lease *Lease, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task == nil {
		return nil
	}

	task := lease.Task.clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["queue.dead_letter.reason"] = reason

	b.mu.Lock()
	defer b.mu.Unlock()

	b.deadLetter[task.Queue] = append(b.deadLetter[task.Queue], task)
	return nil
}

// DeadLetters returns a copy of dead-lettered tasks for queue.
func (b *MemoryQueue) DeadLetters(queue string) []*Task {
	if queue == "" {
		queue = DefaultQueue
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	tasks := b.deadLetter[queue]
	cloned := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		cloned = append(cloned, task.clone())
	}
	return cloned
}
