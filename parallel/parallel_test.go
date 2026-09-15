package parallel_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/parallel/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunExecutesTasksConcurrently(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	started := make(chan struct{}, 3)
	release := make(chan struct{})
	result := make(chan error, 1)
	tasks := make([]parallel.Task, 3)
	for index := range tasks {
		tasks[index] = func(ctx context.Context) error {
			started <- struct{}{}
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return context.Cause(ctx)
			}
		}
	}

	go func() {
		result <- parallel.Run(ctx, tasks...)
	}()

	for range tasks {
		select {
		case <-started:
		case <-ctx.Done():
			require.FailNow(t, "tasks did not start concurrently")
		}
	}

	close(release)
	require.NoError(t, <-result)
}

func TestRunLimitBoundsConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const limit = 2
		releases := []chan struct{}{make(chan struct{}), make(chan struct{})}
		unblocks := make([]func(), len(releases))
		for i, release := range releases {
			unblocks[i] = sync.OnceFunc(func() { close(release) })
		}
		defer func() {
			for _, unblock := range unblocks {
				unblock()
			}
			synctest.Wait()
		}()
		started := make(chan int32, 4)
		result := make(chan error, 1)
		var active atomic.Int32
		tasks := make([]parallel.Task, 4)
		for index := range tasks {
			tasks[index] = func(context.Context) error {
				current := active.Add(1)
				defer active.Add(-1)
				started <- current
				<-releases[index/limit]

				return nil
			}
		}

		go func() {
			result <- parallel.RunLimit(t.Context(), limit, tasks...)
		}()

		for batch, unblock := range unblocks {
			synctest.Wait()
			assert.Len(t, started, (batch+1)*limit)
			assert.Equal(t, int32(limit), active.Load())
			require.Empty(t, result, "RunLimit returned before all batches completed")
			unblock()
		}

		synctest.Wait()
		require.Len(t, result, 1)
		require.NoError(t, <-result)
		require.Len(t, started, len(tasks))
		for range tasks {
			assert.LessOrEqual(t, <-started, int32(limit))
		}
		assert.Zero(t, active.Load())
	})
}

func TestRunReturnsFirstErrorAndCancelsSiblings(t *testing.T) {
	wantErr := errors.New("task failed")
	siblingStarted := make(chan struct{})
	siblingCanceled := make(chan struct{})

	err := parallel.Run(
		t.Context(),
		func(context.Context) error {
			<-siblingStarted

			return wantErr
		},
		func(ctx context.Context) error {
			close(siblingStarted)
			<-ctx.Done()
			close(siblingCanceled)

			return context.Cause(ctx)
		},
	)

	require.ErrorIs(t, err, wantErr)
	assertClosed(t, siblingCanceled)
}

func TestRunObservesParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var called atomic.Bool
	err := parallel.Run(ctx, func(context.Context) error {
		called.Store(true)

		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.False(t, called.Load())
}

func TestRunTrustsSuccessfulTaskResultsAfterAllTasksStart(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{}, 2)
	release := make(chan error, 2)
	result := make(chan error, 1)
	task := func(context.Context) error {
		started <- struct{}{}

		return <-release
	}

	go func() {
		result <- parallel.Run(ctx, task, task)
	}()

	requireStarted(t, t.Context(), started, 2)
	cancel()
	release <- nil
	release <- nil
	require.NoError(t, <-result)
}

func TestRunValidatesInputs(t *testing.T) {
	t.Run("invalid limit", func(t *testing.T) {
		err := parallel.RunLimit(t.Context(), 0)

		require.ErrorIs(t, err, parallel.ErrInvalidLimit)
	})

	t.Run("nil task", func(t *testing.T) {
		err := parallel.Run(t.Context(), nil)

		require.ErrorIs(t, err, parallel.ErrNilTask)
		assert.Contains(t, err.Error(), "index 0")
	})
}

func requireStarted(t *testing.T, ctx context.Context, started <-chan struct{}, count int) {
	t.Helper()

	for range count {
		select {
		case <-started:
		case <-ctx.Done():
			require.FailNow(t, "tasks did not start", context.Cause(ctx))
		}
	}
}

func assertClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	default:
		assert.Fail(t, "channel is not closed")
	}
}
