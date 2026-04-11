package jet

import (
	"context"
	"slices"
)

type Handler func(ctx context.Context, service, method string, request any) (response any, err error)

type Middleware func(Handler) Handler

// Chain chains the middlewares.
//
//	Chain(m1, m2, m3)(xxx) => m1(m2(m3(xxx))
func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for _, mw := range slices.Backward(m) {
			next = mw(next)
		}
		return next
	}
}
