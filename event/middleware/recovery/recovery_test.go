package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/go-fries/fries/event/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userEvent struct {
	name string
}

func TestNewRecoversPanicAsPanicError(t *testing.T) {
	dispatcher := event.New(event.WithMiddleware(New()))
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic("boom")
	})))

	err := dispatcher.Dispatch(t.Context(), userEvent{name: "alice"})
	var panicError *PanicError
	require.ErrorAs(t, err, &panicError)
	assert.Equal(t, "boom", panicError.Value)
	assert.Contains(t, string(panicError.Stack), "recovery_test.go")
	assert.Equal(t, "event recovery: panic: boom", panicError.Error())
}

func TestPanicErrorUnwrapsRecoveredError(t *testing.T) {
	recovered := errors.New("boom")
	dispatcher := event.New(event.WithMiddleware(New()))
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic(recovered)
	})))

	err := dispatcher.Dispatch(t.Context(), userEvent{})
	assert.ErrorIs(t, err, recovered)
}

func TestPanicPropagatesWithoutRecoveryMiddleware(t *testing.T) {
	dispatcher := event.New()
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic("boom")
	})))

	assert.Panics(t, func() { _ = dispatcher.Dispatch(t.Context(), userEvent{}) })
}

func TestWithStackSize(t *testing.T) {
	dispatcher := event.New(event.WithMiddleware(New(WithStackSize(64))))
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic("boom")
	})))

	err := dispatcher.Dispatch(t.Context(), userEvent{})
	var panicError *PanicError
	require.ErrorAs(t, err, &panicError)
	assert.NotEmpty(t, panicError.Stack)
	assert.LessOrEqual(t, len(panicError.Stack), 64)
}

func TestNewIgnoresNilOptionsAndInvalidStackSize(t *testing.T) {
	dispatcher := event.New(event.WithMiddleware(New(nil, WithStackSize(0))))
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic("boom")
	})))

	err := dispatcher.Dispatch(t.Context(), userEvent{})
	var panicError *PanicError
	require.ErrorAs(t, err, &panicError)
	assert.Greater(t, len(panicError.Stack), 64)
}

func TestOuterMiddlewareObservesPanicError(t *testing.T) {
	var observed error
	logging := func(next event.Next) event.Next {
		return func(ctx context.Context, value any) error {
			err := next(ctx, value)
			observed = err
			return err
		}
	}
	dispatcher := event.New(event.WithMiddleware(logging, New()))
	dispatcher.Subscribe(event.HandlerFor[userEvent](event.HandlerFunc[userEvent](func(context.Context, userEvent) error {
		panic("boom")
	})))

	err := dispatcher.Dispatch(t.Context(), userEvent{})
	var panicError *PanicError
	require.ErrorAs(t, observed, &panicError)
	assert.Same(t, err, observed)
}

func TestPanicErrorUnwrapReturnsNilForNonErrorValue(t *testing.T) {
	panicError := &PanicError{Value: "boom"}
	assert.NoError(t, panicError.Unwrap())
}
