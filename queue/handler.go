package queue

import (
	"context"
	"slices"
)

type Handler interface {
	Handle(ctx context.Context, task *Task) error
}

type HandlerFunc func(ctx context.Context, task *Task) error

func (f HandlerFunc) Handle(ctx context.Context, task *Task) error {
	return f(ctx, task)
}

type Middleware func(Handler) Handler

func chain(handler Handler, middleware []Middleware) Handler {
	for _, m := range slices.Backward(middleware) {
		handler = m(handler)
	}
	return handler
}
