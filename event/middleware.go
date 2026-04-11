package event

import (
	"context"
	"slices"
)

type Handler func(ctx context.Context, event any) error

type Middleware func(Handler) Handler

func Chain(mws ...Middleware) Middleware {
	return func(h Handler) Handler {
		for _, mw := range slices.Backward(mws) {
			h = mw(h)
		}
		return h
	}
}
