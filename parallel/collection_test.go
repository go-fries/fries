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
