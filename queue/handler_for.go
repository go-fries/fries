package queue

import (
	"context"
	"errors"
)

// HandlerFor is a type-safe handler interface
type HandlerFor[T any] interface {
	Handle(ctx context.Context, payload T, job *Job) error
}

// HandlerFuncFor is a function adapter that implements Handler interface
// It automatically converts the payload to type T before calling the handler
type HandlerFuncFor[T any] func(ctx context.Context, payload T, job *Job) error

// Handle implements Handler interface, allowing direct use with Manager.Register
func (f HandlerFuncFor[T]) Handle(ctx context.Context, job *Job) error {
	jobFor, err := JobAs[T](job)
	if err != nil {
		return err
	}

	payload, err := jobFor.Payload()
	if err != nil {
		return err
	}

	return f(ctx, payload, job)
}

// Ensure HandlerFuncFor implements Handler interface
var _ Handler = HandlerFuncFor[any](nil)

// MultiHandler combines multiple handlers, trying each until one succeeds
// Useful for queues that handle multiple payload types
type MultiHandler []Handler

// Handle tries each handler until one succeeds (returns nil or non-type-mismatch error)
func (m MultiHandler) Handle(ctx context.Context, job *Job) error {
	var lastErr error

	for _, h := range m {
		err := h.Handle(ctx, job)
		if err == nil {
			return nil
		}

		// If it's a type mismatch error, try next handler
		if errors.Is(err, ErrPayloadTypeMismatch) {
			lastErr = err
			continue
		}

		// For other errors, return immediately
		return err
	}

	// All handlers failed with type mismatch
	if lastErr != nil {
		return lastErr
	}

	return errors.New("queue: no handler matched the job")
}

// NewMultiHandler creates a MultiHandler from multiple handlers
func NewMultiHandler(handlers ...Handler) MultiHandler {
	return MultiHandler(handlers)
}
