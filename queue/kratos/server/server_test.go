package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-fries/fries/queue/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingQueue struct {
	once    sync.Once
	started chan struct{}
}

func newBlockingQueue() *blockingQueue {
	return &blockingQueue{
		started: make(chan struct{}),
	}
}

func (q *blockingQueue) Enqueue(context.Context, *queue.Task) error {
	return nil
}

func (q *blockingQueue) NewConsumer(context.Context, queue.ConsumerConfig) (queue.Consumer, error) {
	return blockingConsumer{queue: q}, nil
}

type dequeueErrorQueue struct {
	err error
}

func (q dequeueErrorQueue) Enqueue(context.Context, *queue.Task) error {
	return nil
}

func (q dequeueErrorQueue) NewConsumer(context.Context, queue.ConsumerConfig) (queue.Consumer, error) {
	return nil, q.err
}

type singleTaskQueue struct {
	mu   sync.Mutex
	task *queue.Task
}

func newSingleTaskQueue(task *queue.Task) *singleTaskQueue {
	return &singleTaskQueue{
		task: task,
	}
}

func (q *singleTaskQueue) Enqueue(context.Context, *queue.Task) error {
	return nil
}

func (q *singleTaskQueue) NewConsumer(context.Context, queue.ConsumerConfig) (queue.Consumer, error) {
	return &singleTaskConsumer{queue: q}, nil
}

type blockingConsumer struct {
	queue *blockingQueue
}

func (c blockingConsumer) Receive(ctx context.Context) (queue.Delivery, error) {
	c.queue.once.Do(func() {
		close(c.queue.started)
	})
	<-ctx.Done()
	return nil, ctx.Err()
}

func (blockingConsumer) Close() error {
	return nil
}

type singleTaskConsumer struct {
	queue *singleTaskQueue
}

func (c *singleTaskConsumer) Receive(ctx context.Context) (queue.Delivery, error) {
	q := c.queue
	q.mu.Lock()
	task := q.task
	q.task = nil
	q.mu.Unlock()
	if task != nil {
		return noopDelivery{task: task}, nil
	}

	<-ctx.Done()
	return nil, ctx.Err()
}

func (c *singleTaskConsumer) Close() error {
	return nil
}

type noopDelivery struct {
	task *queue.Task
}

func (d noopDelivery) Task() *queue.Task {
	return d.task
}

func (noopDelivery) Ack(context.Context) error {
	return nil
}

func (noopDelivery) Retry(context.Context, time.Duration) error {
	return nil
}

func (noopDelivery) DeadLetter(context.Context, string) error {
	return nil
}

func TestServer_StopCancelsWorker(t *testing.T) {
	t.Parallel()

	q := newBlockingQueue()
	server := New(queue.NewWorker(q))

	errs := make(chan error, 1)
	go func() {
		errs <- server.Start(t.Context())
	}()

	select {
	case <-q.started:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for worker start")
	}

	stopCtx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.NoError(t, server.Stop(stopCtx))
	require.NoError(t, <-errs)
}

func TestServer_StopDrainsInFlightTask(t *testing.T) {
	t.Parallel()

	handlerCtxs := make(chan context.Context, 1)
	release := make(chan struct{})
	worker := queue.NewWorker(
		newSingleTaskQueue(&queue.Task{Type: "slow"}),
		queue.Handle("slow", queue.HandlerFunc(func(ctx context.Context, _ *queue.Task) error {
			handlerCtxs <- ctx
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})),
	)
	server := New(worker)

	errs := make(chan error, 1)
	go func() {
		errs <- server.Start(t.Context())
	}()

	var handlerCtx context.Context
	select {
	case handlerCtx = <-handlerCtxs:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for handler start")
	}
	stopErrs := make(chan error, 1)
	go func() {
		stopErrs <- server.Stop(t.Context())
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

func TestServer_StopCancelsInFlightTaskAfterContextDeadline(t *testing.T) {
	t.Parallel()

	handlerStarted := make(chan struct{})
	handlerDone := make(chan error, 1)
	worker := queue.NewWorker(
		newSingleTaskQueue(&queue.Task{Type: "slow"}),
		queue.Handle("slow", queue.HandlerFunc(func(ctx context.Context, _ *queue.Task) error {
			close(handlerStarted)
			<-ctx.Done()
			err := ctx.Err()
			handlerDone <- err
			return err
		})),
	)
	server := New(worker)

	errs := make(chan error, 1)
	go func() {
		errs <- server.Start(t.Context())
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		require.Fail(t, "timeout waiting for handler start")
	}
	stopCtx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()
	require.ErrorIs(t, server.Stop(stopCtx), context.DeadlineExceeded)
	require.ErrorIs(t, <-handlerDone, context.Canceled)
	<-errs
}

func TestServer_StopBeforeStartIsNoop(t *testing.T) {
	t.Parallel()

	server := New(queue.NewWorker(newBlockingQueue()))

	require.NoError(t, server.Stop(t.Context()))
}

func TestServer_StartReturnsWorkerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("dequeue failed")
	server := New(queue.NewWorker(dequeueErrorQueue{err: wantErr}))

	err := server.Start(t.Context())

	require.ErrorIs(t, err, wantErr)
	assert.NoError(t, server.Stop(t.Context()))
}
