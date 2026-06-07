package queue_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-fries/fries/queue/v3"
)

func ExampleNewWorker() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	q := newExampleQueue()
	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		queue.Handle("send_email", queue.HandlerFunc(func(_ context.Context, task *queue.Task) error {
			fmt.Printf("%s: %s\n", task.Type, task.Payload)
			return nil
		})),
		queue.WithPollInterval(time.Millisecond),
	)

	if _, err := producer.Enqueue(ctx, "send_email", []byte("hello")); err != nil {
		panic(err)
	}
	if err := worker.Run(ctx); err != nil {
		panic(err)
	}

	// Output:
	// send_email: hello
}

type sendEmailPayload struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

func ExampleEnqueueFor() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	q := newExampleQueue()
	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		queue.HandleFor("send_email", queue.HandlerFuncFor[sendEmailPayload](func(_ context.Context, task *queue.TaskFor[sendEmailPayload]) error {
			fmt.Printf("%d: %s\n", task.Payload.UserID, task.Payload.Subject)
			return nil
		})),
		queue.WithPollInterval(time.Millisecond),
	)

	_, err := queue.EnqueueFor(ctx, producer, "send_email", sendEmailPayload{
		UserID:  100,
		Subject: "welcome",
	})
	if err != nil {
		panic(err)
	}
	if err := worker.Run(ctx); err != nil {
		panic(err)
	}

	// Output:
	// 100: welcome
}

type exampleQueue struct {
	mu    sync.Mutex
	tasks []*queue.Task
}

func newExampleQueue() *exampleQueue {
	return &exampleQueue{}
}

func (q *exampleQueue) Enqueue(ctx context.Context, task *queue.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	task = task.Clone()
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks = append(q.tasks, task)
	return nil
}

func (q *exampleQueue) Dequeue(ctx context.Context, queueName string, _ time.Duration) (queue.Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if queueName == "" {
		queueName = queue.DefaultQueue
	}

	now := time.Now().UTC()
	q.mu.Lock()
	defer q.mu.Unlock()

	for i, task := range q.tasks {
		if task.Queue != queueName || task.AvailableAt.After(now) {
			continue
		}

		q.tasks = append(q.tasks[:i], q.tasks[i+1:]...)
		task = task.Clone()
		task.Attempt++
		return queue.NewLease(task), nil
	}

	return nil, queue.ErrNoTask
}

func (q *exampleQueue) Ack(ctx context.Context, _ queue.Lease) error {
	return ctx.Err()
}

func (q *exampleQueue) Retry(ctx context.Context, lease queue.Lease, delay time.Duration) error {
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

func (q *exampleQueue) DeadLetter(ctx context.Context, _ queue.Lease, _ string) error {
	return ctx.Err()
}
