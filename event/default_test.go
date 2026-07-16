package event

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDispatcherFunctions(t *testing.T) {
	previous := Default()
	dispatcher := New()
	SetDefault(dispatcher)
	t.Cleanup(func() { SetDefault(previous) })

	assert.Same(t, dispatcher, Default())

	var calls atomic.Int64
	subscription := Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			calls.Add(1)
			return nil
		})),
	)
	require.NoError(t, Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, calls.Load())
	assert.True(t, subscription.Unsubscribe())
}

func TestSetDefaultPanicsForNilDispatcher(t *testing.T) {
	assert.Panics(t, func() { SetDefault(nil) })
}

func TestSetDefaultDoesNotMoveExistingSubscriptions(t *testing.T) {
	previous := Default()
	first := New()
	SetDefault(first)
	t.Cleanup(func() { SetDefault(previous) })

	var calls atomic.Int64
	subscription := Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			calls.Add(1)
			return nil
		})),
	)
	defer subscription.Unsubscribe()

	second := New()
	SetDefault(second)
	require.NoError(t, Dispatch(t.Context(), userEvent{}))
	assert.Zero(t, calls.Load())

	require.NoError(t, first.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, calls.Load())
}

func TestDefaultDispatcherConcurrentAccess(t *testing.T) {
	previous := Default()
	t.Cleanup(func() { SetDefault(previous) })

	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			SetDefault(New())
			assert.NotNil(t, Default())
		})
	}
	workers.Wait()
}
