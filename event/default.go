package event

import (
	"context"
	"sync/atomic"
)

var defaultDispatcher atomic.Pointer[Dispatcher]

func init() {
	defaultDispatcher.Store(New())
}

// Default returns the current default [Dispatcher].
func Default() *Dispatcher {
	return defaultDispatcher.Load()
}

// SetDefault replaces the default [Dispatcher]. It should normally be called
// during application startup, before package-level Subscribe or Dispatch calls.
// SetDefault panics if dispatcher is nil. Existing subscriptions remain bound
// to the Dispatcher that created them.
func SetDefault(dispatcher *Dispatcher) {
	if dispatcher == nil {
		panic("event: nil dispatcher")
	}
	defaultDispatcher.Store(dispatcher)
}

// Subscribe registers listeners on the default [Dispatcher].
func Subscribe(listeners ...Listener) *Subscription {
	return Default().Subscribe(listeners...)
}

// Dispatch synchronously dispatches value through the default [Dispatcher].
func Dispatch(
	ctx context.Context,
	value any,
	options ...DispatchOption,
) error {
	return Default().Dispatch(ctx, value, options...)
}
