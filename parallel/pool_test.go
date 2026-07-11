package parallel_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/parallel/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPoolConfiguration(t *testing.T) {
	t.Run("non-positive workers panic", func(t *testing.T) {
		assert.PanicsWithValue(t, "parallel: worker count must be greater than zero", func() {
			parallel.NewPool(0)
		})
	})

	t.Run("negative queue size keeps default", func(t *testing.T) {
		pool := parallel.NewPool(1, parallel.WithQueueSize(-1))

		require.NotNil(t, pool)
		require.NoError(t, pool.Shutdown(t.Context()))
	})
}

func TestPoolUnbufferedQueueAppliesBackpressure(t *testing.T) {
	pool := requirePool(t, 1, parallel.WithQueueSize(0))
	started := make(chan struct{})
	release := make(chan struct{})

	first, err := pool.Submit(t.Context(), func(context.Context) error {
		close(started)
		<-release

		return nil
	})
	require.NoError(t, err)
	<-started

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	second, err := pool.Submit(ctx, func(context.Context) error { return nil })
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, second)

	close(release)
	require.NoError(t, first.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolValidatesAndReturnsTaskErrors(t *testing.T) {
	pool := requirePool(t, 1)

	future, err := pool.Submit(t.Context(), nil)
	require.ErrorIs(t, err, parallel.ErrNilTask)
	assert.Nil(t, future)

	wantErr := errors.New("task failed")
	future, err = pool.Submit(t.Context(), func(context.Context) error { return wantErr })
	require.NoError(t, err)
	require.ErrorIs(t, future.Wait(t.Context()), wantErr)
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolSubmitRunsAsynchronously(t *testing.T) {
	pool := requirePool(t, 1, parallel.WithQueueSize(0))
	started := make(chan struct{})
	release := make(chan struct{})

	future, err := pool.Submit(t.Context(), func(context.Context) error {
		close(started)
		<-release

		return nil
	})
	require.NoError(t, err)
	<-started

	select {
	case <-future.Done():
		require.FailNow(t, "task completed before it was released")
	default:
	}

	close(release)
	require.NoError(t, future.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolExecuteWaitsForCompletion(t *testing.T) {
	pool := requirePool(t, 1)
	started := make(chan struct{})
	release := make(chan struct{})
	result := make(chan error, 1)

	go func() {
		result <- pool.Execute(t.Context(), func(context.Context) error {
			close(started)
			<-release

			return nil
		})
	}()

	<-started
	select {
	case <-result:
		require.FailNow(t, "Execute returned before the task completed")
	default:
	}

	close(release)
	require.NoError(t, <-result)
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolAppliesQueueBackpressure(t *testing.T) {
	pool := requirePool(t, 1, parallel.WithQueueSize(1))
	started := make(chan struct{})
	release := make(chan struct{}, 2)
	task := func(ctx context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}

	first, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)
	<-started
	second, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	third, err := pool.Submit(ctx, task)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Nil(t, third)

	release <- struct{}{}
	release <- struct{}{}
	require.NoError(t, first.Wait(t.Context()))
	require.NoError(t, second.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestFutureWaitCancellationDoesNotCancelTask(t *testing.T) {
	pool := requirePool(t, 1)
	release := make(chan struct{})

	future, err := pool.Submit(t.Context(), func(context.Context) error {
		<-release

		return nil
	})
	require.NoError(t, err)

	waitContext, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, future.Wait(waitContext), context.Canceled)

	close(release)
	require.NoError(t, future.Wait(t.Context()))
	require.NoError(t, future.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolShutdownDrainsAcceptedTasks(t *testing.T) {
	pool := requirePool(t, 1, parallel.WithQueueSize(1))
	started := make(chan struct{})
	release := make(chan struct{}, 2)
	task := func(context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release

		return nil
	}

	first, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)
	<-started
	second, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)

	waitContext, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, pool.Shutdown(waitContext), context.Canceled)

	future, err := pool.Submit(t.Context(), task)
	require.ErrorIs(t, err, parallel.ErrPoolClosed)
	assert.Nil(t, future)

	release <- struct{}{}
	release <- struct{}{}
	require.NoError(t, first.Wait(t.Context()))
	require.NoError(t, second.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolShutdownUnblocksWaitingSubmit(t *testing.T) {
	pool := requirePool(t, 1, parallel.WithQueueSize(1))
	started := make(chan struct{})
	release := make(chan struct{}, 2)
	task := func(ctx context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}

	first, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)
	<-started
	second, err := pool.Submit(t.Context(), task)
	require.NoError(t, err)

	type submission struct {
		future *parallel.Future
		err    error
	}
	third := make(chan submission, 1)
	go func() {
		future, err := pool.Submit(t.Context(), task)
		third <- submission{future: future, err: err}
	}()

	select {
	case <-third:
		require.FailNow(t, "Submit did not apply backpressure")
	case <-time.After(50 * time.Millisecond):
	}

	waitContext, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, pool.Shutdown(waitContext), context.Canceled)
	got := <-third
	require.ErrorIs(t, got.err, parallel.ErrPoolClosed)
	assert.Nil(t, got.future)

	release <- struct{}{}
	release <- struct{}{}
	require.NoError(t, first.Wait(t.Context()))
	require.NoError(t, second.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestTaskContextCancellationStopsRunningAndQueuedTasks(t *testing.T) {
	wantErr := errors.New("service stopped")
	taskContext, cancelTasks := context.WithCancelCause(t.Context())
	pool := requirePool(t, 1, parallel.WithQueueSize(1))
	started := make(chan struct{})

	running, err := pool.Submit(taskContext, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()

		return context.Cause(ctx)
	})
	require.NoError(t, err)
	<-started

	var queuedCalled atomic.Bool
	queued, err := pool.Submit(taskContext, func(context.Context) error {
		queuedCalled.Store(true)

		return nil
	})
	require.NoError(t, err)

	cancelTasks(wantErr)
	require.ErrorIs(t, running.Wait(t.Context()), wantErr)
	require.ErrorIs(t, queued.Wait(t.Context()), wantErr)
	assert.False(t, queuedCalled.Load())

	future, err := pool.Submit(t.Context(), func(context.Context) error { return nil })
	require.NoError(t, err)
	require.NoError(t, future.Wait(t.Context()))
	require.NoError(t, pool.Shutdown(t.Context()))
}

func TestPoolConcurrentSubmitAndShutdown(t *testing.T) {
	pool := requirePool(t, 4, parallel.WithQueueSize(4))
	const submissions = 100
	type submission struct {
		future *parallel.Future
		err    error
	}
	results := make(chan submission, submissions)
	start := make(chan struct{})
	var submitters sync.WaitGroup

	for range submissions {
		submitters.Go(func() {
			<-start
			future, err := pool.Submit(t.Context(), func(context.Context) error { return nil })
			results <- submission{future: future, err: err}
		})
	}

	close(start)
	require.NoError(t, pool.Shutdown(t.Context()))
	submitters.Wait()
	close(results)

	for result := range results {
		if result.err != nil {
			require.ErrorIs(t, result.err, parallel.ErrPoolClosed)

			continue
		}

		require.NoError(t, result.future.Wait(t.Context()))
	}
}

func requirePool(t *testing.T, workers int, options ...parallel.PoolOption) *parallel.Pool {
	t.Helper()

	return parallel.NewPool(workers, options...)
}
