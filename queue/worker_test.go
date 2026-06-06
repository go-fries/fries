package queue

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkerProcessesAndAcksTask(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	backend := NewMemoryBackend()
	handled := make(chan *Task, 1)
	worker := NewWorker(
		backend,
		Handle("send_email", HandlerFunc(func(_ context.Context, task *Task) error {
			handled <- task.Clone()
			return nil
		})),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	_, err := NewProducer(backend).Enqueue(t.Context(), "send_email", []byte("hello"))
	if err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	select {
	case task := <-handled:
		if string(task.Payload) != "hello" {
			t.Fatalf("payload = %q, want hello", task.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for task")
	}

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("worker run: %v", err)
	}

	if _, err := backend.Dequeue(t.Context(), DefaultQueue, time.Minute); !errors.Is(err, ErrNoTask) {
		t.Fatalf("dequeue after ack error = %v, want %v", err, ErrNoTask)
	}
}

func TestWorkerRetriesThenDeadLetters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	backend := NewMemoryBackend()
	seen := make(chan int, 2)
	worker := NewWorker(
		backend,
		Handle("fail", HandlerFunc(func(_ context.Context, task *Task) error {
			seen <- task.Attempt
			return errors.New("temporary failure")
		})),
		WithPollInterval(time.Millisecond),
		WithRetryPolicy(FixedRetry(2, 0)),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	_, err := NewProducer(backend).Enqueue(t.Context(), "fail", nil)
	if err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	for range 2 {
		select {
		case <-seen:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for retry attempt")
		}
	}

	deadline := time.After(time.Second)
	for len(backend.DeadLetters(DefaultQueue)) != 1 {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for dead letter")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("worker run: %v", err)
	}

	dead := backend.DeadLetters(DefaultQueue)[0]
	if dead.Attempt != 2 {
		t.Fatalf("dead letter attempt = %d, want 2", dead.Attempt)
	}
	if dead.Headers["queue.dead_letter.reason"] == "" {
		t.Fatal("dead letter reason is empty")
	}
}

func TestWorkerConsumesConfiguredQueue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	backend := NewMemoryBackend()
	handled := make(chan struct{}, 1)
	worker := NewWorker(
		backend,
		Handle("custom", HandlerFunc(func(context.Context, *Task) error {
			handled <- struct{}{}
			return nil
		})),
		WithWorkerQueue("critical"),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	_, err := NewProducer(backend).Enqueue(t.Context(), "custom", nil, WithQueue("critical"))
	if err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	select {
	case <-handled:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for task")
	}

	cancel()
	if err := <-errs; err != nil {
		t.Fatalf("worker run: %v", err)
	}
}
