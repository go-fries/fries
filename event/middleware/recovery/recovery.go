package recovery

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/go-fries/fries/event/v3"
)

// HandlerFunc defines a function to handle panic recovery
type HandlerFunc func(ctx context.Context, event any, recovery any, stack []byte) //nolint:gofumpt

// DefaultHandler is the default panic recovery handler that logs the event and stack trace
var DefaultHandler HandlerFunc = func(_ context.Context, event any, recovery any, stack []byte) {
	log.Printf("panic recovery event: %v\nrecovery: %v\nstack trace:\n%s", event, recovery, stack)
}

type options struct {
	handler HandlerFunc
	// stackSize is the size of the stack buffer to allocate
	stackSize int
}

type Option func(*options)

// WithHandler sets a custom panic recovery handler
func WithHandler(h HandlerFunc) Option {
	return func(o *options) {
		if h != nil {
			o.handler = h
		}
	}
}

// WithStackSize sets the size of the stack trace buffer
func WithStackSize(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.stackSize = size
		}
	}
}

// New creates a new recovery middleware
func New(opts ...Option) event.Middleware {
	o := &options{
		handler:   DefaultHandler,
		stackSize: 4 << 10, // nolint:mnd // 4KB default stack size
	}
	for _, opt := range opts {
		opt(o)
	}

	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, event any) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := make([]byte, o.stackSize)
					stack = stack[:runtime.Stack(stack, false)]
					o.handler(ctx, event, r, stack)

					// Convert panic to error
					err = fmt.Errorf("panic recovered: %v", r)
				}
			}()
			return next(ctx, event)
		}
	}
}
