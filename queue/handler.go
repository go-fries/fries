package queue

import "context"

// Handler processes queue jobs
type Handler interface {
	Handle(ctx context.Context, job Job) error
}

// HandlerFunc is a function adapter for Handler
type HandlerFunc func(ctx context.Context, job Job) error

// Handle implements the Handler interface
func (f HandlerFunc) Handle(ctx context.Context, job Job) error {
	return f(ctx, job)
}
