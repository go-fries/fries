package queue

import (
	"context"
	"sync"
	"time"
)

type testQueue struct {
	mu         sync.Mutex
	queues     map[string][]*Task
	deadLetter map[string][]*Task
}

func newTestQueue() *testQueue {
	return &testQueue{
		queues:     make(map[string][]*Task),
		deadLetter: make(map[string][]*Task),
	}
}

func (q *testQueue) Enqueue(ctx context.Context, task *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	task = task.clone()
	if task.Queue == "" {
		task.Queue = DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.queues[task.Queue] = append(q.queues[task.Queue], task)
	return nil
}

func (q *testQueue) Dequeue(ctx context.Context, queueName string, _ time.Duration) (Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queueName == "" {
		queueName = DefaultQueue
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
		task = task.clone()
		task.Attempt++
		return NewLease(task), nil
	}

	return nil, ErrNoTask
}

func (q *testQueue) Ack(ctx context.Context, _ Lease) error {
	return ctx.Err()
}

func (q *testQueue) Retry(ctx context.Context, lease Lease, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	return q.Enqueue(ctx, task)
}

func (q *testQueue) DeadLetter(ctx context.Context, lease Lease, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if lease == nil || lease.Task() == nil {
		return nil
	}

	task := lease.Task().clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["queue.dead_letter.reason"] = reason

	q.mu.Lock()
	defer q.mu.Unlock()

	q.deadLetter[task.Queue] = append(q.deadLetter[task.Queue], task)
	return nil
}

func (q *testQueue) DeadLetters(queueName string) []*Task {
	if queueName == "" {
		queueName = DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tasks := q.deadLetter[queueName]
	cloned := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		cloned = append(cloned, task.clone())
	}
	return cloned
}
