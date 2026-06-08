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

func (q *blockingQueue) Dequeue(ctx context.Context, _ string, _ time.Duration) (queue.Lease, error) {
	q.once.Do(func() {
		close(q.started)
	})
	<-ctx.Done()
	return nil, ctx.Err()
}

func (q *blockingQueue) Ack(context.Context, queue.Lease) error {
	return nil
}

func (q *blockingQueue) Retry(context.Context, queue.Lease, time.Duration) error {
	return nil
}

func (q *blockingQueue) DeadLetter(context.Context, queue.Lease, string) error {
	return nil
}

type dequeueErrorQueue struct {
	err error
}

func (q dequeueErrorQueue) Enqueue(context.Context, *queue.Task) error {
	return nil
}

func (q dequeueErrorQueue) Dequeue(context.Context, string, time.Duration) (queue.Lease, error) {
	return nil, q.err
}

func (q dequeueErrorQueue) Ack(context.Context, queue.Lease) error {
	return nil
}

func (q dequeueErrorQueue) Retry(context.Context, queue.Lease, time.Duration) error {
	return nil
}

func (q dequeueErrorQueue) DeadLetter(context.Context, queue.Lease, string) error {
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
