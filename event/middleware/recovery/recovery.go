package recovery

import (
	"context"
	"fmt"
	"runtime"

	"github.com/go-fries/fries/event/v4"
)

// PanicError describes a panic recovered while handling an event.
type PanicError struct {
	Value any
	Stack []byte
}

// Error returns a description of the recovered panic.
func (e *PanicError) Error() string {
	return fmt.Sprintf("event recovery: panic: %v", e.Value)
}

// Unwrap returns the recovered value when it is an error.
func (e *PanicError) Unwrap() error {
	err, _ := e.Value.(error)
	return err
}

// New creates middleware that converts panics into PanicError values. New does
// not log, emit metrics, or report alerts; those concerns belong to outer
// middleware or the Dispatch caller.
func New(options ...Option) event.Middleware {
	c := newConfig(options...)
	return func(next event.AnyHandler) event.AnyHandler {
		return func(ctx context.Context, value any) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					stack := make([]byte, c.stackSize)
					stack = stack[:runtime.Stack(stack, false)]
					err = &PanicError{Value: recovered, Stack: stack}
				}
			}()
			return next(ctx, value)
		}
	}
}
