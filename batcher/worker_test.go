package batcher_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-fries/fries/batcher/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerBatchSizeAndSliceOwnership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var batches [][]int
		b := newBatcher(t, func(_ context.Context, items []int) error {
			batches = append(batches, items)

			return nil
		}, batcher.WithBatchSize(2), batcher.WithFlushInterval(time.Hour))
		for i := range 5 {
			require.NoError(t, b.Add(t.Context(), i))
		}
		synctest.Wait()
		assert.Equal(t, [][]int{{0, 1}, {2, 3}}, batches)
		require.NoError(t, b.Shutdown(t.Context()))
		assert.Equal(t, [][]int{{0, 1}, {2, 3}, {4}}, batches)
	})
}

func TestWorkerDefaultBatchSize(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var sizes []int
		b := newBatcher(t, func(_ context.Context, items []int) error {
			sizes = append(sizes, len(items))

			return nil
		})
		for i := range 100 {
			require.NoError(t, b.Add(t.Context(), i))
		}
		synctest.Wait()
		assert.Equal(t, []int{100}, sizes)
	})
}

func TestWorkerIntervalStartsWithFirstItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		batches := make(chan []int, 10)
		b := newBatcher(t, func(_ context.Context, items []int) error {
			batches <- items

			return nil
		})
		time.Sleep(10 * time.Second)
		synctest.Wait()
		assert.Empty(t, batches)
		require.NoError(t, b.Add(t.Context(), 1))
		synctest.Wait()
		time.Sleep(600 * time.Millisecond)
		require.NoError(t, b.Add(t.Context(), 2))
		synctest.Wait()
		assert.Empty(t, batches)
		time.Sleep(400 * time.Millisecond)
		synctest.Wait()
		require.Len(t, batches, 1)
		assert.Equal(t, []int{1, 2}, <-batches)
		time.Sleep(5 * time.Second)
		synctest.Wait()
		assert.Empty(t, batches)
		require.NoError(t, b.Flush(t.Context()))
		require.NoError(t, b.Shutdown(t.Context()))
		assert.Empty(t, batches)
	})
}

func TestWorkerTimerResetsAfterFlush(t *testing.T) {
	for _, manual := range []bool{false, true} {
		name := "full batch"
		if manual {
			name = "manual flush"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				batches := make(chan []int, 10)
				b := newBatcher(t, func(_ context.Context, items []int) error {
					batches <- items

					return nil
				}, batcher.WithBatchSize(2))
				require.NoError(t, b.Add(t.Context(), 1))
				synctest.Wait()
				time.Sleep(600 * time.Millisecond)
				if manual {
					require.NoError(t, b.Flush(t.Context()))
				} else {
					require.NoError(t, b.Add(t.Context(), 2))
				}
				synctest.Wait()
				require.Len(t, batches, 1)
				<-batches
				require.NoError(t, b.Add(t.Context(), 3))
				synctest.Wait()
				time.Sleep(400 * time.Millisecond)
				synctest.Wait()
				assert.Empty(t, batches)
				time.Sleep(600 * time.Millisecond)
				synctest.Wait()
				require.Len(t, batches, 1)
				assert.Equal(t, []int{3}, <-batches)
			})
		})
	}
}

func TestWorkerErrorsRemainVisibleAndProcessingContinues(t *testing.T) {
	firstErr := errors.New("first failure")
	secondErr := errors.New("second failure")
	var results []batcher.Result
	b := newBatcher(t, func(_ context.Context, items []int) error {
		switch items[0] {
		case 1:
			return firstErr
		case 2:
			return secondErr
		default:
			return nil
		}
	}, batcher.WithBatchSize(1), batcher.WithOnResult(func(result batcher.Result) {
		results = append(results, result)
	}))
	require.NoError(t, b.Flush(t.Context()))
	require.NoError(t, b.Add(t.Context(), 1))
	require.ErrorIs(t, b.Flush(t.Context()), firstErr)
	require.NoError(t, b.Add(t.Context(), 2))
	require.NoError(t, b.Add(t.Context(), 3))
	require.ErrorIs(t, b.Flush(t.Context()), firstErr)
	require.ErrorIs(t, b.Shutdown(t.Context()), firstErr)
	require.ErrorIs(t, b.Shutdown(t.Context()), firstErr)
	assert.Equal(t, []batcher.Result{
		{Size: 1, Err: firstErr},
		{Size: 1, Err: secondErr},
		{Size: 1},
	}, results)
}

func TestWorkerResultCallbackIsPartOfCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		b := newBatcher(t, func(context.Context, []int) error { return nil },
			batcher.WithBatchSize(1),
			batcher.WithOnResult(func(batcher.Result) { <-release }),
		)
		unblock := cleanupRelease(t, release)
		require.NoError(t, b.Add(t.Context(), 1))
		flushed := make(chan error, 1)
		go func() { flushed <- b.Flush(t.Context()) }()
		synctest.Wait()
		select {
		case err := <-flushed:
			t.Fatalf("Flush returned before the result callback finished: %v", err)
		default:
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, b.Shutdown(ctx), context.Canceled)
		unblock()
		require.NoError(t, <-flushed)
		require.NoError(t, b.Shutdown(t.Context()))
	})
}
