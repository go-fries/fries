package recovery

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"github.com/go-fries/fries/queue/v3"
)

// HandlerFunc handles a recovered queue handler panic.
type HandlerFunc func(ctx context.Context, task *queue.Task, recovered any, stack []byte)

// DefaultHandler logs recovered panics and stack traces.
var DefaultHandler HandlerFunc = func(_ context.Context, task *queue.Task, recovered any, stack []byte) {
	if task == nil {
		log.Printf("queue panic recovered: task=<nil> recovered=%v\nstack trace:\n%s", recovered, stack)
		return
	}
	log.Printf(
		"queue panic recovered: task_id=%s task_type=%s queue=%s attempt=%d recovered=%v\nstack trace:\n%s",
		task.ID,
		task.Type,
		task.Queue,
		task.Attempt,
		recovered,
		stack,
	)
}

type config struct {
	handler   HandlerFunc
	stackSize int
}

// Option configures recovery middleware.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithHandler sets a custom recovery handler.
func WithHandler(handler HandlerFunc) Option {
	return optionFunc(func(c *config) {
		if handler != nil {
			c.handler = handler
		}
	})
}

// WithStackSize sets the stack trace buffer size.
func WithStackSize(size int) Option {
	return optionFunc(func(c *config) {
		if size > 0 {
			c.stackSize = size
		}
	})
}

// New creates queue panic recovery middleware.
func New(opts ...Option) queue.Middleware {
	c := newConfig(opts...)

	return func(next queue.Handler) queue.Handler {
		return queue.HandlerFunc(func(ctx context.Context, task *queue.Task) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					stack := make([]byte, c.stackSize)
					stack = stack[:runtime.Stack(stack, false)]
					c.handler(ctx, task, recovered, stack)
					err = fmt.Errorf("queue handler panic recovered: %v", recovered)
				}
			}()
			return next.Handle(ctx, task)
		})
	}
}

func newConfig(opts ...Option) *config {
	c := &config{
		handler:   DefaultHandler,
		stackSize: 4 << 10, //nolint:mnd // 4KB default stack size.
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}
