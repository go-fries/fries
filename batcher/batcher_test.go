package batcher_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/batcher/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatcherBackpressure(t *testing.T) {
	for _, capacity := range []int{0, 1} {
		t.Run(strconv.Itoa(capacity), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				release := make(chan struct{})
				var processed []int
				b := newBatcher(t, func(_ context.Context, items []int) error {
					<-release
					processed = append(processed, items...)

					return nil
				}, batcher.WithBatchSize(1), batcher.WithQueueSize(capacity))
				unblock := cleanupRelease(t, release)
				require.NoError(t, b.Add(t.Context(), 1))
				synctest.Wait()
				if capacity > 0 {
					require.NoError(t, b.TryAdd(2))
				}
				require.ErrorIs(t, b.TryAdd(3), batcher.ErrFull)
				ctx, cancel := context.WithCancelCause(t.Context())
				defer cancel(nil)
				result := make(chan error, 1)
				go func() { result <- b.Add(ctx, 3) }()
				synctest.Wait()
				select {
				case err := <-result:
					t.Fatalf("Add returned while capacity was unavailable: %v", err)
				default:
				}
				wantErr := errors.New("admission canceled")
				cancel(wantErr)
				require.ErrorIs(t, <-result, wantErr)
				go func() { result <- b.Add(t.Context(), 4) }()
				synctest.Wait()
				assert.Empty(t, result)
				unblock()
				require.NoError(t, <-result)
				require.NoError(t, b.Shutdown(t.Context()))
				if capacity > 0 {
					assert.Equal(t, []int{1, 2, 4}, processed)
				} else {
					assert.Equal(t, []int{1, 4}, processed)
				}
			})
		})
	}
}

func TestBatcherTryAddUnbuffered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var processed []int
		b := newBatcher(t, func(_ context.Context, items []int) error {
			processed = append(processed, items...)

			return nil
		}, batcher.WithQueueSize(0))
		synctest.Wait()
		require.NoError(t, b.TryAdd(1))
		require.NoError(t, b.Flush(t.Context()))
		assert.Equal(t, []int{1}, processed)
	})
}

func TestBatcherFlushBoundary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		firstRelease := make(chan struct{})
		lastRelease := make(chan struct{})
		lastErr := errors.New("failure after the flush barrier")
		var batches [][]int
		b := newBatcher(t, func(_ context.Context, items []int) error {
			switch items[0] {
			case 1:
				<-firstRelease
			case 4:
				<-lastRelease
			}
			batches = append(batches, items)
			if items[0] == 4 {
				return lastErr
			}

			return nil
		}, batcher.WithBatchSize(2), batcher.WithFlushInterval(time.Hour))
		unblockFirst := cleanupRelease(t, firstRelease)
		unblockLast := cleanupRelease(t, lastRelease)
		for _, item := range []int{1, 2, 3} {
			require.NoError(t, b.Add(t.Context(), item))
		}
		flushed := make(chan error, 1)
		go func() { flushed <- b.Flush(t.Context()) }()
		synctest.Wait()
		require.NoError(t, b.Add(t.Context(), 4))
		require.NoError(t, b.Add(t.Context(), 5))
		unblockFirst()
		synctest.Wait()
		select {
		case err := <-flushed:
			require.NoError(t, err)
		default:
			t.Fatal("Flush waited for items accepted after its barrier")
		}
		assert.Equal(t, [][]int{{1, 2}, {3}}, batches)
		unblockLast()
		require.ErrorIs(t, b.Shutdown(t.Context()), lastErr)
		assert.Equal(t, [][]int{{1, 2}, {3}, {4, 5}}, batches)
	})
}

func TestBatcherFlushCancellationPreservesWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var processed []int
		b := newBatcher(t, func(ctx context.Context, items []int) error {
			<-release
			assert.NoError(t, context.Cause(ctx))
			processed = append(processed, items...)

			return nil
		}, batcher.WithBatchSize(1))
		unblock := cleanupRelease(t, release)
		addCtx, cancelAdd := context.WithCancel(t.Context())
		require.NoError(t, b.Add(addCtx, 1))
		cancelAdd()
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)
		result := make(chan error, 1)
		go func() { result <- b.Flush(ctx) }()
		synctest.Wait()
		wantErr := errors.New("wait canceled")
		cancel(wantErr)
		require.ErrorIs(t, <-result, wantErr)
		unblock()
		require.NoError(t, b.Flush(t.Context()))
		assert.Equal(t, []int{1}, processed)
	})
}

func TestBatcherShutdownUnblocksAdmissionsAndDrains(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var processed []int
		b := newBatcher(t, func(_ context.Context, items []int) error {
			<-release
			processed = append(processed, items...)

			return nil
		}, batcher.WithBatchSize(1), batcher.WithQueueSize(1))
		unblock := cleanupRelease(t, release)
		require.NoError(t, b.Add(t.Context(), 1))
		synctest.Wait()
		require.NoError(t, b.Add(t.Context(), 2))
		added := make(chan error, 1)
		flushed := make(chan error, 1)
		go func() { added <- b.Add(t.Context(), 3) }()
		go func() { flushed <- b.Flush(t.Context()) }()
		synctest.Wait()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, b.Shutdown(ctx), context.Canceled)
		require.ErrorIs(t, <-added, batcher.ErrClosed)
		require.ErrorIs(t, <-flushed, batcher.ErrClosed)
		require.ErrorIs(t, b.Add(t.Context(), 4), batcher.ErrClosed)
		require.ErrorIs(t, b.TryAdd(4), batcher.ErrClosed)
		require.ErrorIs(t, b.Flush(t.Context()), batcher.ErrClosed)
		unblock()
		require.NoError(t, b.Shutdown(t.Context()))
		require.NoError(t, b.Shutdown(ctx))
		assert.Equal(t, []int{1, 2}, processed)
	})
}

func TestBatcherProcessingCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)
		var processed []int
		b, err := batcher.New(ctx, func(ctx context.Context, items []int) error {
			<-ctx.Done()
			processed = append(processed, items...)

			return context.Cause(ctx)
		}, batcher.WithBatchSize(1))
		require.NoError(t, err)
		require.NoError(t, b.Add(t.Context(), 1))
		require.NoError(t, b.Add(t.Context(), 2))
		synctest.Wait()
		wantErr := errors.New("processing stopped")
		cancel(wantErr)
		synctest.Wait()
		require.ErrorIs(t, b.TryAdd(3), batcher.ErrClosed)
		require.ErrorIs(t, b.Shutdown(t.Context()), wantErr)
		assert.Equal(t, []int{1, 2}, processed)
	})
}

func TestBatcherValidatesContexts(t *testing.T) {
	b := newBatcher(t, func(context.Context, []int) error { return nil })
	var nilContext context.Context
	require.ErrorIs(t, b.Add(nilContext, 1), batcher.ErrInvalidContext)
	require.ErrorIs(t, b.Flush(nilContext), batcher.ErrInvalidContext)
	require.ErrorIs(t, b.Shutdown(nilContext), batcher.ErrInvalidContext)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, b.Add(ctx, 1), context.Canceled)
	require.ErrorIs(t, b.Flush(ctx), context.Canceled)
	require.NoError(t, b.Add(t.Context(), 2))
	require.NoError(t, b.Shutdown(t.Context()))
}

func TestBatcherConcurrentAdmissionsFlushAndShutdown(t *testing.T) {
	var processed []int
	b := newBatcher(t, func(_ context.Context, items []int) error {
		processed = append(processed, items...)

		return nil
	}, batcher.WithBatchSize(7), batcher.WithQueueSize(8))
	const count = 200
	accepted := make(chan int, count+1)
	start := make(chan struct{})
	var callers sync.WaitGroup
	for i := range count {
		callers.Go(func() {
			<-start
			var err error
			if i%2 == 0 {
				err = b.Add(t.Context(), i)
			} else {
				err = b.TryAdd(i)
			}
			if err == nil {
				accepted <- i
			} else {
				assert.True(t, errors.Is(err, batcher.ErrClosed) || errors.Is(err, batcher.ErrFull))
			}
		})
	}
	for range 10 {
		callers.Go(func() {
			<-start
			err := b.Flush(t.Context())
			assert.True(t, err == nil || errors.Is(err, batcher.ErrClosed))
		})
		callers.Go(func() {
			<-start
			assert.NoError(t, b.Shutdown(t.Context()))
		})
	}
	// Ensure the test always includes accepted work, even if shutdown wins the race.
	require.NoError(t, b.Add(t.Context(), -1))
	accepted <- -1
	close(start)
	callers.Wait()
	close(accepted)
	require.NoError(t, b.Shutdown(t.Context()))
	want := make([]int, 0, count)
	for item := range accepted {
		want = append(want, item)
	}
	assert.ElementsMatch(t, want, processed)
}

func newBatcher[T any](t *testing.T, handler batcher.Handler[T], options ...batcher.Option) *batcher.Batcher[T] {
	t.Helper()
	b, err := batcher.New(t.Context(), handler, options...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = b.Shutdown(context.WithoutCancel(t.Context())) })

	return b
}

func cleanupRelease(t *testing.T, release chan struct{}) func() {
	t.Helper()
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)

	return unblock
}
