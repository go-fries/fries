package parallel_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/go-fries/fries/parallel/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForEachProcessesEveryValue(t *testing.T) {
	var (
		mu     sync.Mutex
		values []int
	)

	err := parallel.ForEach(
		t.Context(), 2, []int{1, 2, 3, 4},
		func(_ context.Context, value int) error {
			mu.Lock()
			defer mu.Unlock()

			values = append(values, value)

			return nil
		},
	)

	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2, 3, 4}, values)
}

func TestMapPreservesInputOrder(t *testing.T) {
	results, err := parallel.Map(
		t.Context(), 3, []int{3, 1, 2},
		func(_ context.Context, value int) (int, error) {
			return value * 10, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []int{30, 10, 20}, results)
}

func TestMapReturnsNoPartialResultsOnFailure(t *testing.T) {
	wantErr := errors.New("map failed")

	results, err := parallel.Map(
		t.Context(), 2, []int{1, 2, 3},
		func(_ context.Context, value int) (int, error) {
			if value == 2 {
				return 0, wantErr
			}

			return value * 10, nil
		},
	)

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, results)
}

func TestMapResultsKeepsPerItemOutcomes(t *testing.T) {
	wantErr := errors.New("item failed")

	results, err := parallel.MapResults(
		t.Context(), 2, []int{1, 2, 3},
		func(_ context.Context, value int) (int, error) {
			if value == 2 {
				return 0, wantErr
			}

			return value * 10, nil
		},
	)

	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, 10, results[0].Value)
	assert.NoError(t, results[0].Err)
	assert.Zero(t, results[1].Value)
	assert.ErrorIs(t, results[1].Err, wantErr)
	assert.Equal(t, 30, results[2].Value)
	assert.NoError(t, results[2].Err)
}

func TestMapResultsReturnsPartialOutcomesOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	type output struct {
		results []parallel.Result[int]
		err     error
	}
	result := make(chan output, 1)

	go func() {
		results, err := parallel.MapResults(
			ctx, 1, []int{1, 2, 3},
			func(ctx context.Context, value int) (int, error) {
				close(started)
				<-ctx.Done()

				return value, context.Cause(ctx)
			},
		)
		result <- output{results: results, err: err}
	}()

	<-started
	cancel()
	got := <-result

	require.ErrorIs(t, got.err, context.Canceled)
	require.Len(t, got.results, 3)
	for _, item := range got.results {
		assert.ErrorIs(t, item.Err, context.Canceled)
	}
}

func TestFilterPreservesInputOrder(t *testing.T) {
	values, err := parallel.Filter(
		t.Context(), 3, []int{5, 2, 4, 1, 3},
		func(_ context.Context, value int) (bool, error) {
			return value%2 == 1, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []int{5, 1, 3}, values)
}

func TestFilterReturnsNoPartialResultsOnFailure(t *testing.T) {
	wantErr := errors.New("predicate failed")

	values, err := parallel.Filter(
		t.Context(), 2, []int{1, 2, 3},
		func(_ context.Context, value int) (bool, error) {
			if value == 2 {
				return false, wantErr
			}

			return true, nil
		},
	)

	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, values)
}

func TestCollectionHelpersValidateInputs(t *testing.T) {
	t.Run("ForEach limit", func(t *testing.T) {
		err := parallel.ForEach(
			t.Context(), -1, []int{1},
			func(context.Context, int) error { return nil },
		)

		require.ErrorIs(t, err, parallel.ErrInvalidLimit)
	})

	t.Run("ForEach function", func(t *testing.T) {
		err := parallel.ForEach[int](t.Context(), 1, nil, nil)

		require.ErrorIs(t, err, parallel.ErrNilFunc)
	})

	t.Run("Map limit", func(t *testing.T) {
		_, err := parallel.Map(
			t.Context(), 0, []int{1},
			func(context.Context, int) (int, error) { return 0, nil },
		)

		require.ErrorIs(t, err, parallel.ErrInvalidLimit)
	})

	t.Run("Map function", func(t *testing.T) {
		_, err := parallel.Map[int, int](t.Context(), 1, nil, nil)

		require.ErrorIs(t, err, parallel.ErrNilFunc)
	})

	t.Run("MapResults function", func(t *testing.T) {
		_, err := parallel.MapResults[int, int](t.Context(), 1, nil, nil)

		require.ErrorIs(t, err, parallel.ErrNilFunc)
	})

	t.Run("Filter function", func(t *testing.T) {
		_, err := parallel.Filter[int](t.Context(), 1, nil, nil)

		require.ErrorIs(t, err, parallel.ErrNilFunc)
	})
}

func TestMapEmptyInput(t *testing.T) {
	results, err := parallel.Map(
		t.Context(), 1, []int{},
		func(_ context.Context, value int) (int, error) { return value, nil },
	)

	require.NoError(t, err)
	assert.Empty(t, results)
	assert.NotNil(t, results)
}
