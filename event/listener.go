package event

import (
	"context"
)

type Listener[T any] interface {
	Handle(ctx context.Context, event T) error
}

type AnyListener interface {
	Handle(ctx context.Context, event any) error
}

// listenerAdapter adapts a Listener[T] to an AnyListener.
type listenerAdapter[T any] struct {
	L Listener[T]
}

// AdaptListener adapts a Listener[T] to an AnyListener.
func AdaptListener[T any](l Listener[T]) AnyListener {
	return &listenerAdapter[T]{L: l}
}

func (a *listenerAdapter[T]) Handle(ctx context.Context, event any) error {
	if e, ok := event.(T); ok {
		return a.L.Handle(ctx, e)
	}
	return nil
}

// ListenerFunc is a function type that implements the Listener interface.
type ListenerFunc[T any] func(ctx context.Context, event T) error

var _ Listener[string] = ListenerFunc[string](nil)

func (f ListenerFunc[T]) Handle(ctx context.Context, event T) error {
	return f(ctx, event)
}

// AdaptListenerFunc adapts a function to the Listener interface.
func AdaptListenerFunc[T any](f func(ctx context.Context, event T) error) AnyListener {
	return AdaptListener(ListenerFunc[T](f))
}
