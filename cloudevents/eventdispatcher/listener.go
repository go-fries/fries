package eventdispatcher

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

var ErrFailedToUnmarshalEvent = fmt.Errorf("failed to unmarshal event data")

type Listener interface {
	Handle(ctx context.Context, event cloudevents.Event) error
}

type AnyListener[T any] interface {
	Handle(ctx context.Context, event T) error
}

type listenerAdapter[T any] struct {
	l AnyListener[T]
}

func ListenerAdapter[T any](l AnyListener[T]) Listener {
	return &listenerAdapter[T]{l: l}
}

func (a *listenerAdapter[T]) Handle(ctx context.Context, event cloudevents.Event) error {
	var data T
	if err := event.DataAs(&data); err != nil {
		return ErrFailedToUnmarshalEvent
	}
	return a.l.Handle(ctx, data)
}

type ListenerFunc[T any] func(ctx context.Context, event T) error

var _ AnyListener[cloudevents.Event] = ListenerFunc[cloudevents.Event](nil)

func (l ListenerFunc[T]) Handle(ctx context.Context, event cloudevents.Event) error {
	var data T
	if err := event.DataAs(&data); err != nil {
		return ErrFailedToUnmarshalEvent
	}
	return l(ctx, data)
}
