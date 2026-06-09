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

func (q dequeueErrorQueue) NewConsumer(context.Context, ConsumerConfig) (Consumer, error) {
	return nil, q.err
}

type receiveErrorQueue struct {
	err error
}

func (q receiveErrorQueue) Enqueue(context.Context, *Task) error {
	return nil
}

func (q receiveErrorQueue) NewConsumer(context.Context, ConsumerConfig) (Consumer, error) {
	return receiveErrorConsumer(q), nil
}

type receiveErrorConsumer struct {
	err error
}

func (c receiveErrorConsumer) Receive(context.Context) (Delivery, error) {
	return nil, c.err
}

func (receiveErrorConsumer) Close() error {
	return nil
}

type recordingDelivery struct {
	task       *Task
	ack        func(context.Context) error
	retry      func(context.Context, time.Duration) error
	deadLetter func(context.Context, string) error
}

func (d *recordingDelivery) Task() *Task {
	return d.task
}

func (d *recordingDelivery) Ack(ctx context.Context) error {
	if d.ack != nil {
		return d.ack(ctx)
	}
	return nil
}

func (d *recordingDelivery) Retry(ctx context.Context, delay time.Duration) error {
	if d.retry != nil {
		return d.retry(ctx, delay)
	}
	return nil
}

func (d *recordingDelivery) DeadLetter(ctx context.Context, reason string) error {
	if d.deadLetter != nil {
		return d.deadLetter(ctx, reason)
	}
	return nil
}

func TestWorker_ConfigDefaults(t *testing.T) {
	t.Parallel()

	config := newWorkerConfig(
		WithQueue(""),
		WithConsumerName(""),
		WithConcurrency(0),
		WithHandlerTimeout(0),
		WithRetryPolicy(nil),
		WithMiddleware(),
		Handle("", HandlerFunc(func(context.Context, *Task) error { return nil })),
		Handle("ignored", nil),
	)

	assert.Equal(t, DefaultQueue, config.queue)
	assert.Empty(t, config.consumerName)
	assert.Equal(t, 1, config.concurrency)
	assert.Zero(t, config.handlerTimeout)
	assert.Equal(t, defaultSettlementTimeout, config.settlementTimeout)
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
		WithQueue("critical"),
		WithConsumerName("worker-1"),
		WithConcurrency(4),
		WithHandlerTimeout(time.Second),
		WithSettlementTimeout(2*time.Second),
		WithRetryPolicy(retryPolicy),
		WithMiddleware(middleware),
		Handle("send_email", handler),
	)

	assert.Equal(t, "critical", config.queue)
	assert.Equal(t, "worker-1", config.consumerName)
	assert.Equal(t, 4, config.concurrency)
	assert.Equal(t, time.Second, config.handlerTimeout)
	assert.Equal(t, 2*time.Second, config.settlementTimeout)
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

	require.Empty(t, q.queues[DefaultQueue])
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
		WithQueue("critical"),
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

func TestWorker_RunReturnsUnexpectedConsumerClosed(t *testing.T) {
	t.Parallel()

	worker := NewWorker(receiveErrorQueue{err: ErrConsumerClosed})

	err := worker.Run(t.Context())

	require.ErrorIs(t, err, ErrConsumerClosed)
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
	delivery := &testDelivery{queue: q, task: &Task{
		ID:    "task-1",
		Type:  "unknown",
		Queue: DefaultQueue,
	}}

	err := worker.process(t.Context(), delivery)
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

func TestWorker_HandlerTimeoutRetriesWithSettlementContext(t *testing.T) {
	t.Parallel()

	retried := make(chan time.Duration, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("slow", HandlerFunc(func(ctx context.Context, _ *Task) error {
			<-ctx.Done()
			return ctx.Err()
		})),
		WithHandlerTimeout(time.Millisecond),
		WithRetryPolicy(FixedRetry(2, 0)),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "slow", Attempt: 1},
		retry: func(ctx context.Context, delay time.Duration) error {
			require.NoError(t, ctx.Err())
			retried <- delay
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, time.Duration(0), <-retried)
}

func TestWorker_HandlerTimeoutDeadLettersWithSettlementContext(t *testing.T) {
	t.Parallel()

	deadLettered := make(chan string, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("slow", HandlerFunc(func(ctx context.Context, _ *Task) error {
			<-ctx.Done()
			return ctx.Err()
		})),
		WithHandlerTimeout(time.Millisecond),
		WithRetryPolicy(NoRetry()),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "slow", Attempt: 1},
		deadLetter: func(ctx context.Context, reason string) error {
			require.NoError(t, ctx.Err())
			deadLettered <- reason
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Contains(t, <-deadLettered, ErrRetryExhausted.Error())
}

func TestWorker_SettlementTimeout(t *testing.T) {
	t.Parallel()

	worker := NewWorker(
		newTestQueue(),
		Handle("ok", HandlerFunc(func(context.Context, *Task) error {
			return nil
		})),
		WithSettlementTimeout(time.Millisecond),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "ok"},
		ack: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	err := worker.process(t.Context(), delivery)

	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWorker_DiscardsTask(t *testing.T) {
	t.Parallel()

	acked := make(chan struct{}, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("discard", HandlerFunc(func(context.Context, *Task) error {
			return ErrDiscard
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "discard"},
		ack: func(context.Context) error {
			acked <- struct{}{}
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Len(t, acked, 1)
}

func TestWorker_DeadLetterErrorDeadLettersTask(t *testing.T) {
	t.Parallel()

	deadLettered := make(chan string, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("invalid", HandlerFunc(func(context.Context, *Task) error {
			return DeadLetter("invalid payload")
		})),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "invalid"},
		deadLetter: func(_ context.Context, reason string) error {
			deadLettered <- reason
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, "invalid payload", <-deadLettered)
}

func TestWorker_RetryAfterOverridesRetryDelay(t *testing.T) {
	t.Parallel()

	retried := make(chan time.Duration, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("rate_limited", HandlerFunc(func(context.Context, *Task) error {
			return RetryAfter(5 * time.Second)
		})),
		WithRetryPolicy(FixedRetry(2, time.Second)),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "rate_limited", Attempt: 1},
		retry: func(_ context.Context, delay time.Duration) error {
			retried <- delay
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Equal(t, 5*time.Second, <-retried)
}

func TestWorker_RetryAfterRespectsRetryPolicyBudget(t *testing.T) {
	t.Parallel()

	deadLettered := make(chan string, 1)
	worker := NewWorker(
		newTestQueue(),
		Handle("rate_limited", HandlerFunc(func(context.Context, *Task) error {
			return RetryAfter(time.Second)
		})),
		WithRetryPolicy(FixedRetry(1, 0)),
	)
	delivery := &recordingDelivery{
		task: &Task{Type: "rate_limited", Attempt: 1},
		deadLetter: func(_ context.Context, reason string) error {
			deadLettered <- reason
			return nil
		},
	}

	err := worker.process(t.Context(), delivery)

	require.NoError(t, err)
	assert.Contains(t, <-deadLettered, ErrRetryExhausted.Error())
}

func TestWorker_RunContextCancellationCancelsInFlightTask(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

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
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(ctx)
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for handler start")
	}

	cancel()
	require.ErrorIs(t, <-handlerDone, context.Canceled)
	require.NoError(t, <-errs)
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
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(t.Context())
	}()

	var handlerCtx context.Context
	select {
	case handlerCtx = <-handlerCtxs:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for handler start")
	}
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
	)

	errs := make(chan error, 1)
	go func() {
		errs <- worker.Run(t.Context())
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for handler start")
	}
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
