package event

import (
	"context"
	"slices"
)

// Next handles one matching event in a middleware chain.
type Next func(context.Context, any) error

// Middleware wraps a Next function.
type Middleware func(Next) Next

func chain(middleware ...Middleware) Middleware {
	return func(next Next) Next {
		for _, item := range slices.Backward(middleware) {
			if item != nil {
				next = item(next)
			}
		}
		return next
	}
}
