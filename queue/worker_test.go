package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerProcessesAndAcksTask(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := NewMemoryQueue()
	handled := make(chan *Task, 1)
	worker := NewWorker(
		q,
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

	_, err := NewProducer(q).Enqueue(t.Context(), "send_email", []byte("hello"))
	require.NoError(t, err)

	select {
	case task := <-handled:
		assert.Equal(t, "hello", string(task.Payload))
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for task")
	}

	cancel()
	require.NoError(t, <-errs)

	_, err = q.Dequeue(t.Context(), DefaultQueue, time.Minute)
	require.ErrorIs(t, err, ErrNoTask)
}

func TestWorkerRetriesThenDeadLetters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := NewMemoryQueue()
	seen := make(chan int, 2)
	worker := NewWorker(
		q,
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

	_, err := NewProducer(q).Enqueue(t.Context(), "fail", nil)
	require.NoError(t, err)

	for range 2 {
		select {
		case <-seen:
		case <-time.After(time.Second):
			require.Fail(t, "timeout waiting for retry attempt")
		}
	}

	deadline := time.After(time.Second)
	for len(q.DeadLetters(DefaultQueue)) != 1 {
		select {
		case <-deadline:
			require.Fail(t, "timeout waiting for dead letter")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	cancel()
	require.NoError(t, <-errs)

	dead := q.DeadLetters(DefaultQueue)[0]
	assert.Equal(t, 2, dead.Attempt)
	assert.NotEmpty(t, dead.Metadata["queue.dead_letter.reason"])
}

func TestWorkerConsumesConfiguredQueue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := NewMemoryQueue()
	handled := make(chan struct{}, 1)
	worker := NewWorker(
		q,
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

	_, err := NewProducer(q).Enqueue(t.Context(), "custom", nil, WithQueue("critical"))
	require.NoError(t, err)

	select {
	case <-handled:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for task")
	}

	cancel()
	require.NoError(t, <-errs)
}
