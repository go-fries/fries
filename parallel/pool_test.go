package parallel_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
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
	testPoolBackpressure(t, 0)
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

func TestPoolTrustsSuccessfulTaskResultAfterContextCancellation(t *testing.T) {
	pool := requirePool(t, 1)
	taskContext, cancel := context.WithCancel(t.Context())

	future, err := pool.Submit(taskContext, func(context.Context) error {
		cancel()

		return nil
	})
	require.NoError(t, err)
	require.NoError(t, future.Wait(t.Context()))
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
	synctest.Test(t, func(t *testing.T) {
		pool := requirePool(t, 1)
		started := make(chan struct{})
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer func() {
			unblock()
			assert.NoError(t, pool.Shutdown(context.WithoutCancel(t.Context())))
			synctest.Wait()
		}()
		result := make(chan error, 1)
		wantErr := errors.New("task failed")

		go func() {
			result <- pool.Execute(t.Context(), func(context.Context) error {
				close(started)
				<-release

				return wantErr
			})
		}()

		synctest.Wait()
		assertClosed(t, started)
		require.Empty(t, result, "Execute returned before the task completed")

		unblock()
		synctest.Wait()
		require.Len(t, result, 1)
		require.ErrorIs(t, <-result, wantErr)
		require.NoError(t, pool.Shutdown(t.Context()))
	})
}

func TestPoolAppliesQueueBackpressure(t *testing.T) {
	testPoolBackpressure(t, 1)
}

func testPoolBackpressure(t *testing.T, queueSize int) {
	t.Helper()

	synctest.Test(t, func(t *testing.T) {
		const timeout = time.Second
		pool := requirePool(t, 1, parallel.WithQueueSize(queueSize))
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer func() {
			unblock()
			assert.NoError(t, pool.Shutdown(context.WithoutCancel(t.Context())))
			synctest.Wait()
		}()
		var calls atomic.Int32
		task := func(context.Context) error {
			calls.Add(1)
			<-release

			return nil
		}
		first, err := pool.Submit(t.Context(), task)
		require.NoError(t, err)
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		accepted := make([]*parallel.Future, 1, 1+queueSize)
		accepted[0] = first
		for range queueSize {
			future, err := pool.Submit(t.Context(), task)
			require.NoError(t, err)
			accepted = append(accepted, future)
		}

		type submission struct {
			future *parallel.Future
			err    error
		}
		result := make(chan submission, 1)
		var rejectedCalled atomic.Bool
		ctx, cancel := context.WithTimeout(t.Context(), timeout)
		defer cancel()
		started := time.Now()
		go func() {
			future, err := pool.Submit(ctx, func(context.Context) error {
				rejectedCalled.Store(true)

				return nil
			})
			result <- submission{future: future, err: err}
		}()

		synctest.Wait()
		require.Empty(t, result, "Submit did not apply backpressure")
		time.Sleep(timeout - time.Nanosecond)
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
		require.Empty(t, result, "Submit returned before its deadline")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Len(t, result, 1, "deadline did not unblock Submit")
		got := <-result
		require.ErrorIs(t, got.err, context.DeadlineExceeded)
		assert.Nil(t, got.future)
		assert.Equal(t, timeout, time.Since(started))

		unblock()
		for _, future := range accepted {
			require.NoError(t, future.Wait(t.Context()))
		}
		require.NoError(t, pool.Shutdown(t.Context()))
		assert.Equal(t, int32(1+queueSize), calls.Load())
		assert.False(t, rejectedCalled.Load())
	})
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
	started := make(chan struct{}, 1)
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
	synctest.Test(t, func(t *testing.T) {
		pool := requirePool(t, 1, parallel.WithQueueSize(1))
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		defer func() {
			unblock()
			assert.NoError(t, pool.Shutdown(context.WithoutCancel(t.Context())))
			synctest.Wait()
		}()
		var calls atomic.Int32
		task := func(ctx context.Context) error {
			calls.Add(1)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return context.Cause(ctx)
			}
		}

		first, err := pool.Submit(t.Context(), task)
		require.NoError(t, err)
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load())
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

		synctest.Wait()
		require.Empty(t, third, "Submit did not apply backpressure")
		assert.Equal(t, int32(1), calls.Load())

		waitContext, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, pool.Shutdown(waitContext), context.Canceled)
		synctest.Wait()
		require.Len(t, third, 1, "Shutdown did not unblock Submit")
		got := <-third
		require.ErrorIs(t, got.err, parallel.ErrPoolClosed)
		assert.Nil(t, got.future)

		shutdown := make(chan error, 1)
		go func() {
			shutdown <- pool.Shutdown(t.Context())
		}()
		synctest.Wait()
		require.Empty(t, shutdown, "Shutdown returned before accepted tasks completed")
		assert.Equal(t, int32(1), calls.Load())

		unblock()
		synctest.Wait()
		require.NoError(t, first.Wait(t.Context()))
		require.NoError(t, second.Wait(t.Context()))
		require.Len(t, shutdown, 1)
		require.NoError(t, <-shutdown)
		assert.Equal(t, int32(2), calls.Load())
	})
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

func TestPoolTrySubmitCapacity(t *testing.T) {
	for _, size := range []int{0, 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			pool := requirePool(t, 1, parallel.WithQueueSize(size))
			release := make(chan struct{})
			started := make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(func() {
				unblock()
				require.NoError(t, pool.Shutdown(context.WithoutCancel(t.Context())))
			})
			_, err := pool.Submit(t.Context(), func(context.Context) error {
				close(started)
				<-release

				return nil
			})
			require.NoError(t, err)
			<-started

			wantErr := errors.New("accepted task failed")
			var accepted *parallel.Future
			if size > 0 {
				accepted, err = pool.TrySubmit(t.Context(), func(context.Context) error { return wantErr })
				require.NoError(t, err)
			}

			var called atomic.Bool
			future, err := pool.TrySubmit(t.Context(), func(context.Context) error {
				called.Store(true)

				return nil
			})
			require.ErrorIs(t, err, parallel.ErrPoolFull)
			assert.Nil(t, future)
			unblock()
			require.NoError(t, pool.Shutdown(t.Context()))
			assert.False(t, called.Load())
			if accepted != nil {
				require.ErrorIs(t, accepted.Wait(t.Context()), wantErr)
			}
		})
	}
}

func TestPoolTrySubmitValidation(t *testing.T) {
	pool := requirePool(t, 1)
	task := func(context.Context) error { return nil }
	future, err := pool.TrySubmit(t.Context(), nil)
	require.ErrorIs(t, err, parallel.ErrNilTask)
	assert.Nil(t, future)
	ctx, cancel := context.WithCancelCause(t.Context())
	wantErr := errors.New("stopped")
	cancel(wantErr)
	future, err = pool.TrySubmit(ctx, task)
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, future)
	require.NoError(t, pool.Shutdown(t.Context()))
	future, err = pool.TrySubmit(t.Context(), task)
	require.ErrorIs(t, err, parallel.ErrPoolClosed)
	assert.Nil(t, future)
}

func TestPoolConcurrentTrySubmitAndShutdown(t *testing.T) {
	pool := requirePool(t, 4)
	start := make(chan struct{})
	var submitters sync.WaitGroup
	for range 100 {
		submitters.Go(func() {
			<-start
			future, err := pool.TrySubmit(t.Context(), func(context.Context) error { return nil })
			if err != nil {
				assert.True(t, errors.Is(err, parallel.ErrPoolClosed) || errors.Is(err, parallel.ErrPoolFull))

				return
			}
			assert.NoError(t, future.Wait(t.Context()))
		})
	}
	close(start)
	require.NoError(t, pool.Shutdown(t.Context()))
	submitters.Wait()
}

func requirePool(t *testing.T, workers int, options ...parallel.PoolOption) *parallel.Pool {
	t.Helper()

	return parallel.NewPool(workers, options...)
}
