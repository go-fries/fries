package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dequeueErrorQueue struct {
	err error
}

func (q dequeueErrorQueue) Enqueue(context.Context, *Task) error {
	return nil
}

func (q dequeueErrorQueue) Dequeue(context.Context, string, time.Duration) (Lease, error) {
	return nil, q.err
}

func (q dequeueErrorQueue) Ack(context.Context, Lease) error {
	return nil
}

func (q dequeueErrorQueue) Retry(context.Context, Lease, time.Duration) error {
	return nil
}

func (q dequeueErrorQueue) DeadLetter(context.Context, Lease, string) error {
	return nil
}

func TestWorker_ConfigDefaults(t *testing.T) {
	t.Parallel()

	config := newWorkerConfig(
		WithWorkerQueue(""),
		WithConcurrency(0),
		WithPollInterval(0),
		WithVisibilityTimeout(0),
		WithHandlerTimeout(0),
		WithRetryPolicy(nil),
		WithMiddleware(),
		Handle("", HandlerFunc(func(context.Context, *Task) error { return nil })),
		Handle("ignored", nil),
	)

	assert.Equal(t, DefaultQueue, config.queue)
	assert.Equal(t, 1, config.concurrency)
	assert.Equal(t, time.Second, config.pollInterval)
	assert.Equal(t, 5*time.Minute, config.visibilityTimeout)
	assert.Zero(t, config.handlerTimeout)
	assert.NotNil(t, config.retryPolicy)
	assert.Empty(t, config.middleware)
	assert.Empty(t, config.handlers)
}

func TestWorker_ConfigOptions(t *testing.T) {
	t.Parallel()

	middleware := Middleware(func(next Handler) Handler { return next })
	handler := HandlerFunc(func(context.Context, *Task) error { return nil })
	retryPolicy := NoRetry()

	config := newWorkerConfig(
		WithWorkerQueue("critical"),
		WithConcurrency(4),
		WithPollInterval(10*time.Millisecond),
		WithVisibilityTimeout(30*time.Second),
		WithHandlerTimeout(time.Second),
		WithRetryPolicy(retryPolicy),
		WithMiddleware(middleware),
		Handle("send_email", handler),
	)

	assert.Equal(t, "critical", config.queue)
	assert.Equal(t, 4, config.concurrency)
	assert.Equal(t, 10*time.Millisecond, config.pollInterval)
	assert.Equal(t, 30*time.Second, config.visibilityTimeout)
	assert.Equal(t, time.Second, config.handlerTimeout)
	assert.Equal(t, retryPolicy, config.retryPolicy)
	assert.Len(t, config.middleware, 1)
	assert.NotNil(t, config.handlers["send_email"])
}

func TestWorker_ProcessesAndAcksTask(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := newTestQueue()
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

func TestWorker_RetriesThenDeadLetters(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := newTestQueue()
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

func TestWorker_ConsumesConfiguredQueue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	q := newTestQueue()
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

func TestWorker_RunReturnsQueueErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("dequeue failed")
	worker := NewWorker(dequeueErrorQueue{err: wantErr})

	err := worker.Run(t.Context())

	require.ErrorIs(t, err, wantErr)
}

func TestWorker_RunStopsOnContextQueueErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "canceled", err: context.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			worker := NewWorker(dequeueErrorQueue{err: tt.err})

			err := worker.Run(t.Context())

			require.NoError(t, err)
		})
	}
}

func TestWorker_DeadLettersTaskWithoutHandler(t *testing.T) {
	t.Parallel()

	q := newTestQueue()
	worker := NewWorker(q)
	lease := NewLease(&Task{
		ID:    "task-1",
		Type:  "unknown",
		Queue: DefaultQueue,
	})

	err := worker.process(t.Context(), lease)
	require.NoError(t, err)

	deadLetters := q.DeadLetters(DefaultQueue)
	require.Len(t, deadLetters, 1)
	assert.Equal(t, "task-1", deadLetters[0].ID)
	assert.Equal(t, ErrHandlerNotFound.Error(), deadLetters[0].Metadata["queue.dead_letter.reason"])
}

func TestWorker_HandlerTimeout(t *testing.T) {
	t.Parallel()

	worker := NewWorker(newTestQueue(), WithHandlerTimeout(time.Millisecond))
	err := worker.handle(t.Context(), HandlerFunc(func(ctx context.Context, _ *Task) error {
		<-ctx.Done()
		return ctx.Err()
	}), &Task{})

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWorker_StopDrainsInFlightTask(t *testing.T) {
	t.Parallel()

	q := newTestQueue()
	_, err := NewProducer(q).Enqueue(t.Context(), "slow", nil)
	require.NoError(t, err)

	handlerCtxs := make(chan context.Context, 1)
	release := make(chan struct{})
	worker := NewWorker(
		q,
		Handle("slow", HandlerFunc(func(ctx context.Context, _ *Task) error {
			handlerCtxs <- ctx
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(t.Context())
	}()

	handlerCtx := <-handlerCtxs
	stopErrs := make(chan error, 1)
	go func() {
		stopErrs <- worker.Stop(t.Context())
	}()

	select {
	case err := <-stopErrs:
		require.Failf(t, "stop returned before in-flight task finished", "err=%v", err)
	default:
	}
	select {
	case <-handlerCtx.Done():
		require.Fail(t, "handler context was canceled before drain timeout")
	default:
	}

	close(release)
	require.NoError(t, <-stopErrs)
	require.NoError(t, <-errs)
}

func TestWorker_StopCancelsInFlightTaskAfterContextDeadline(t *testing.T) {
	t.Parallel()

	q := newTestQueue()
	_, err := NewProducer(q).Enqueue(t.Context(), "slow", nil)
	require.NoError(t, err)

	handlerStarted := make(chan struct{})
	handlerDone := make(chan error, 1)
	worker := NewWorker(
		q,
		Handle("slow", HandlerFunc(func(ctx context.Context, _ *Task) error {
			close(handlerStarted)
			<-ctx.Done()
			err := ctx.Err()
			handlerDone <- err
			return err
		})),
		WithPollInterval(time.Millisecond),
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(t.Context())
	}()

	<-handlerStarted
	stopCtx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, worker.Stop(stopCtx), context.DeadlineExceeded)
	require.ErrorIs(t, <-handlerDone, context.Canceled)
	require.ErrorIs(t, <-errs, context.Canceled)
}

func TestWorker_MiddlewareOrder(t *testing.T) {
	t.Parallel()

	var calls []string
	worker := NewWorker(
		newTestQueue(),
		WithMiddleware(
			func(next Handler) Handler {
				return HandlerFunc(func(ctx context.Context, task *Task) error {
					calls = append(calls, "first before")
					err := next.Handle(ctx, task)
					calls = append(calls, "first after")
					return err
				})
			},
			func(next Handler) Handler {
				return HandlerFunc(func(ctx context.Context, task *Task) error {
					calls = append(calls, "second before")
					err := next.Handle(ctx, task)
					calls = append(calls, "second after")
					return err
				})
			},
		),
	)

	err := worker.handle(t.Context(), HandlerFunc(func(context.Context, *Task) error {
		calls = append(calls, "handler")
		return nil
	}), &Task{})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"first before",
		"second before",
		"handler",
		"second after",
		"first after",
	}, calls)
}
