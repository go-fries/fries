package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/queue/v3"
)

// Queue is an in-memory queue implementation for tests and local use.
type Queue struct {
	mu         sync.Mutex
	queues     map[string][]*queue.Task
	deadLetter map[string][]*queue.Task
}

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates an empty in-memory queue.
func NewQueue() *Queue {
	return &Queue{
		queues:     make(map[string][]*queue.Task),
		deadLetter: make(map[string][]*queue.Task),
	}
}

// Enqueue stores task in memory.
func (q *Queue) Enqueue(ctx context.Context, task *queue.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task = task.Clone()
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.queues[task.Queue] = append(q.queues[task.Queue], task)
	return nil
}

// Dequeue returns the first available task from queueName.
func (q *Queue) Dequeue(ctx context.Context, queueName string, _ time.Duration) (queue.Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queueName == "" {
		queueName = queue.DefaultQueue
	}

	now := time.Now().UTC()
	q.mu.Lock()
	defer q.mu.Unlock()

	tasks := q.queues[queueName]
	for i, task := range tasks {
		if task.AvailableAt.After(now) {
			continue
		}

		q.queues[queueName] = append(tasks[:i], tasks[i+1:]...)
		task = task.Clone()
		task.Attempt++
		return queue.NewLease(task), nil
	}

	return nil, queue.ErrNoTask
}

// Ack marks a memory lease as complete.
func (q *Queue) Ack(ctx context.Context, _ queue.Lease) error {
	return ctx.Err()
}

// Retry re-enqueues a leased task after delay.
func (q *Queue) Retry(ctx context.Context, lease queue.Lease, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	return q.Enqueue(ctx, task)
}

// DeadLetter stores a leased task in the in-memory dead-letter list.
func (q *Queue) DeadLetter(ctx context.Context, lease queue.Lease, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().Clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["queue.dead_letter.reason"] = reason

	q.mu.Lock()
	defer q.mu.Unlock()

	q.deadLetter[task.Queue] = append(q.deadLetter[task.Queue], task)
	return nil
}

// DeadLetters returns a copy of dead-lettered tasks for queueName.
func (q *Queue) DeadLetters(queueName string) []*queue.Task {
	if queueName == "" {
		queueName = queue.DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tasks := q.deadLetter[queueName]
	cloned := make([]*queue.Task, 0, len(tasks))
	for _, task := range tasks {
		cloned = append(cloned, task.Clone())
	}
	return cloned
}
