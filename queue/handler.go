package queue

import (
	"context"
	"slices"
)

// Handler processes a task delivered by a worker.
type Handler interface {
	// Handle processes task and returns nil only when the task should be acknowledged.
	Handle(ctx context.Context, task *Task) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, task *Task) error

// Handle calls f(ctx, task).
func (f HandlerFunc) Handle(ctx context.Context, task *Task) error {
	return f(ctx, task)
}

// Middleware wraps a handler with cross-cutting behavior.
type Middleware func(Handler) Handler

func chain(handler Handler, middleware []Middleware) Handler {
	for _, m := range slices.Backward(middleware) {
		handler = m(handler)
	}
	return handler
}
