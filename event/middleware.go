package event

import (
	"context"
	"slices"
)

// AnyHandler invokes one matching event handler through a type-erased
// middleware boundary. It does not subscribe to every event type.
type AnyHandler func(context.Context, any) error

// Middleware wraps an [AnyHandler].
type Middleware func(AnyHandler) AnyHandler

func chain(middleware ...Middleware) Middleware {
	return func(next AnyHandler) AnyHandler {
		for _, item := range slices.Backward(middleware) {
			if item != nil {
				next = item(next)
			}
		}
		return next
	}
}
