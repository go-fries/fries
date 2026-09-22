package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenSubscriptionAndInstanceIsolation(t *testing.T) {
	dispatcher := New()
	other := New()
	var calls []string
	handler := func(_ context.Context, value userEvent) error {
		calls = append(calls, "function:"+value.name)
		return nil
	}
	first := Listen(dispatcher, handler)
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(_ context.Context, value userEvent) error {
		calls = append(calls, "legacy:"+value.name)
		return nil
	})))
	Listen(dispatcher, handler)
	Listen(other, handler)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{name: "alice"}))
	assert.Equal(t, []string{"function:alice", "legacy:alice", "function:alice"}, calls)

	assert.True(t, first.Unsubscribe())
	assert.False(t, first.Unsubscribe())
	calls = nil
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{name: "bob"}))
	assert.Equal(t, []string{"legacy:bob", "function:bob"}, calls)

	calls = nil
	require.NoError(t, other.Dispatch(t.Context(), userEvent{name: "carol"}))
	assert.Equal(t, []string{"function:carol"}, calls)
}

func TestListenExactTypes(t *testing.T) {
	dispatcher := New()
	var values []userEvent
	var pointers []*userEvent
	Listen(dispatcher, func(_ context.Context, value userEvent) error {
		values = append(values, value)
		return nil
	})
	Listen(dispatcher, func(_ context.Context, value *userEvent) error {
		pointers = append(pointers, value)
		return nil
	})

	value := userEvent{name: "alice"}
	require.NoError(t, dispatcher.Dispatch(t.Context(), value))
	require.NoError(t, dispatcher.Dispatch(t.Context(), &value))
	require.NoError(t, dispatcher.Dispatch(t.Context(), (*userEvent)(nil)))
	require.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{}))
	assert.Equal(t, []userEvent{value}, values)
	require.Len(t, pointers, 2)
	assert.Same(t, &value, pointers[0])
	assert.Nil(t, pointers[1])
}

func TestListenMiddlewareAndErrors(t *testing.T) {
	var calls []string
	dispatcher := New(WithMiddleware(func(next AnyHandler) AnyHandler {
		return func(ctx context.Context, value any) error {
			calls = append(calls, "before")
			err := next(ctx, value)
			calls = append(calls, "after")
			return err
		}
	}))
	ctx := t.Context()
	Listen(dispatcher, func(received context.Context, _ userEvent) error {
		assert.Equal(t, ctx, received)
		calls = append(calls, "first")
		return assert.AnError
	})
	Listen(dispatcher, func(context.Context, userEvent) error {
		calls = append(calls, "second")
		return nil
	})

	assert.ErrorIs(t, dispatcher.Dispatch(ctx, userEvent{}), assert.AnError)
	assert.Equal(t, []string{"before", "first", "after"}, calls)

	calls = nil
	assert.ErrorIs(t, dispatcher.Dispatch(ctx, userEvent{}, ContinueOnError()), assert.AnError)
	assert.Equal(t, []string{"before", "first", "after", "before", "second", "after"}, calls)

	calls = nil
	canceled, cancel := context.WithCancelCause(ctx)
	cancel(assert.AnError)
	assert.ErrorIs(t, dispatcher.Dispatch(canceled, userEvent{}), assert.AnError)
	assert.Empty(t, calls)
}

func TestListenValidation(t *testing.T) {
	t.Run("nil dispatcher", func(t *testing.T) {
		assert.PanicsWithValue(t, "event: nil dispatcher", func() {
			Listen(nil, func(context.Context, userEvent) error { return nil })
		})
	})

	t.Run("nil handler", func(t *testing.T) {
		assert.PanicsWithValue(t, "event: nil handler", func() {
			Listen[userEvent](New(), nil)
		})
	})

	t.Run("interface event type", func(t *testing.T) {
		assert.PanicsWithValue(t, "event: handler event type must not be an interface", func() {
			Listen(New(), func(context.Context, any) error { return nil })
		})
	})
}
