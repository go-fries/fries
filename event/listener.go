package event

import (
	"context"
)

type Listener[T any] interface {
	Handle(ctx context.Context, event T) error
}

type AnyListener interface {
	HandleEvent(ctx context.Context, event any) error
}

func AdaptListener[T any](l Listener[T]) AnyListener {
	return &listenerAdapter[T]{L: l}
}

type listenerAdapter[T any] struct {
	L Listener[T]
}

func (a *listenerAdapter[T]) HandleEvent(ctx context.Context, event any) error {
	if e, ok := event.(T); ok {
		return a.L.Handle(ctx, e)
	}
	return nil
}

type ListenerFunc[T any] func(ctx context.Context, event T) error

var _ Listener[string] = ListenerFunc[string](nil)

func (f ListenerFunc[T]) Handle(ctx context.Context, event T) error {
	return f(ctx, event)
}
