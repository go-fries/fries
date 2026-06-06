package queue

import (
	"context"
	"sync"
	"time"
)

type MemoryBackend struct {
	mu         sync.Mutex
	queues     map[string][]*Task
	deadLetter map[string][]*Task
}

var _ Backend = (*MemoryBackend)(nil)

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		queues:     make(map[string][]*Task),
		deadLetter: make(map[string][]*Task),
	}
}

func (b *MemoryBackend) Enqueue(ctx context.Context, task *Task) error {
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

func (b *MemoryBackend) Dequeue(ctx context.Context, queue string, _ time.Duration) (*Lease, error) {
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

func (b *MemoryBackend) Ack(ctx context.Context, _ *Lease) error {
	return ctx.Err()
}

func (b *MemoryBackend) Retry(ctx context.Context, lease *Lease, delay time.Duration) error {
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

func (b *MemoryBackend) DeadLetter(ctx context.Context, lease *Lease, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task == nil {
		return nil
	}

	task := lease.Task.clone()
	if task.Headers == nil {
		task.Headers = make(map[string]string)
	}
	task.Headers["queue.dead_letter.reason"] = reason

	b.mu.Lock()
	defer b.mu.Unlock()

	b.deadLetter[task.Queue] = append(b.deadLetter[task.Queue], task)
	return nil
}

func (b *MemoryBackend) DeadLetters(queue string) []*Task {
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
