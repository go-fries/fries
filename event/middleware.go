package event

import "context"

type Handler func(ctx context.Context, event any) error

type Middleware func(Handler) Handler

func Chain(mws ...Middleware) Middleware {
	return func(h Handler) Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}
