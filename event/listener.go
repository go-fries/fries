package event

import (
	"context"
	"reflect"
)

// Handler handles events of type T.
type Handler[T any] interface {
	Handle(context.Context, T) error
}

// HandlerFunc adapts a function to [Handler].
type HandlerFunc[T any] func(context.Context, T) error

// Handle calls f with ctx and value.
func (f HandlerFunc[T]) Handle(ctx context.Context, value T) error {
	return f(ctx, value)
}

// Listener is a type-aware handler accepted by [Dispatcher.Subscribe]. Listener
// values are created by [HandlerFor].
type Listener interface {
	definition() listenerDefinition
}

type listenerDefinition struct {
	typeOf reflect.Type
	next   AnyHandler
}

type typedListener[T any] struct {
	handler Handler[T]
}

func (l typedListener[T]) definition() listenerDefinition {
	return listenerDefinition{
		typeOf: reflect.TypeFor[T](),
		next: func(ctx context.Context, value any) error {
			return l.handler.Handle(ctx, value.(T))
		},
	}
}

// HandlerFor adapts handler to a [Listener] for the exact concrete type T.
// HandlerFor panics if T is an interface type or handler is nil.
func HandlerFor[T any](handler Handler[T]) Listener {
	typeOf := reflect.TypeFor[T]()
	if typeOf.Kind() == reflect.Interface {
		panic("event: handler event type must not be an interface")
	}
	if isNilHandler(handler) {
		panic("event: nil handler")
	}
	return typedListener[T]{handler: handler}
}

func isNilHandler[T any](handler Handler[T]) bool {
	if handler == nil {
		return true
	}
	value := reflect.ValueOf(handler)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
