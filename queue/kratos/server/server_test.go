package server

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/queue/v4"
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

	synctest.Test(t, func(t *testing.T) {
		ctx, cancelRun := context.WithCancel(t.Context())
		defer func() {
			cancelRun()
			synctest.Wait()
		}()
		q := newBlockingQueue()
		server := New(queue.NewWorker(q))

		errs := make(chan error, 1)
		go func() {
			errs <- server.Start(ctx)
		}()

		synctest.Wait()
		select {
		case <-q.started:
		default:
			t.Fatal("worker has not entered Receive")
		}
		require.Empty(t, errs)

		require.NoError(t, server.Stop(t.Context()))
		synctest.Wait()
		require.Len(t, errs, 1)
		require.NoError(t, <-errs)
	})
}

func TestServer_StopDrainsInFlightTask(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx, cancelRun := context.WithCancel(t.Context())
		defer func() {
			cancelRun()
			synctest.Wait()
		}()
		handlerCtxs := make(chan context.Context, 1)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer unblock()
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
			errs <- server.Start(ctx)
		}()

		synctest.Wait()
		require.Len(t, handlerCtxs, 1)
		handlerCtx := <-handlerCtxs
		stopErrs := make(chan error, 1)
		go func() {
			stopErrs <- server.Stop(t.Context())
		}()

		synctest.Wait()
		require.Empty(t, stopErrs, "Stop returned before the handler completed")
		require.Empty(t, errs)
		assert.NoError(t, handlerCtx.Err())

		unblock()
		synctest.Wait()
		require.Len(t, stopErrs, 1)
		require.Len(t, errs, 1)
		require.NoError(t, <-stopErrs)
		require.NoError(t, <-errs)
	})
}

func TestServer_StopCancelsInFlightTaskAfterContextDeadline(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		ctx, cancelRun := context.WithCancel(t.Context())
		defer func() {
			cancelRun()
			synctest.Wait()
		}()
		handlerStarted := make(chan context.Context, 1)
		handlerDone := make(chan error, 1)
		worker := queue.NewWorker(
			newSingleTaskQueue(&queue.Task{Type: "slow"}),
			queue.Handle("slow", queue.HandlerFunc(func(ctx context.Context, _ *queue.Task) error {
				handlerStarted <- ctx
				<-ctx.Done()
				err := ctx.Err()
				handlerDone <- err
				return err
			})),
		)
		server := New(worker)

		errs := make(chan error, 1)
		go func() {
			errs <- server.Start(ctx)
		}()

		synctest.Wait()
		require.Len(t, handlerStarted, 1)
		handlerCtx := <-handlerStarted
		stopCtx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		stopped := make(chan error, 1)
		go func() { stopped <- server.Stop(stopCtx) }()
		synctest.Wait()
		time.Sleep(time.Second - time.Nanosecond)
		synctest.Wait()
		assert.NoError(t, handlerCtx.Err())
		require.Empty(t, stopped)
		require.Empty(t, handlerDone)
		require.Empty(t, errs)
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, stopped, 1)
		require.ErrorIs(t, <-stopped, context.DeadlineExceeded)
		require.Len(t, handlerDone, 1)
		require.ErrorIs(t, <-handlerDone, context.Canceled)
		require.Len(t, errs, 1)
		require.NoError(t, <-errs)
	})
}

func TestServer_StopBeforeStartIsNoop(t *testing.T) {
	t.Parallel()

	server := New(queue.NewWorker(newBlockingQueue()))

	require.NoError(t, server.Stop(t.Context()))
}

func TestServer_StartReturnsWorkerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("dequeue failed")
	handler := &recordingHandler{}
	server := New(queue.NewWorker(dequeueErrorQueue{err: wantErr}), WithLogger(slog.New(handler)))

	err := server.Start(t.Context())

	require.ErrorIs(t, err, wantErr)
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[Queue] server starting", handler.records[0].Message)
	assert.NoError(t, server.Stop(t.Context()))
}

func TestServer_StopWritesToConfiguredLogger(t *testing.T) {
	t.Parallel()

	handler := &recordingHandler{}
	server := New(queue.NewWorker(newBlockingQueue()), WithLogger(slog.New(handler)))

	require.NoError(t, server.Stop(t.Context()))
	require.Len(t, handler.records, 1)
	assert.Equal(t, slog.LevelInfo, handler.records[0].Level)
	assert.Equal(t, "[Queue] server stopping", handler.records[0].Message)
}

type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *recordingHandler) Handle(_ context.Context, record slog.Record) error {
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *recordingHandler) WithGroup(string) slog.Handler {
	return h
}
