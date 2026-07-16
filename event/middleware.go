package event

import (
	"context"
	"slices"
)

// Next handles one matching event in a middleware chain.
type Next func(context.Context, any) error

// Middleware wraps a Next function.
type Middleware func(Next) Next

// Chain combines middleware in declaration order from outermost to innermost.
// Nil middleware is ignored.
func Chain(middleware ...Middleware) Middleware {
	return func(next Next) Next {
		for _, item := range slices.Backward(middleware) {
			if item != nil {
				next = item(next)
			}
		}
		return next
	}
}
