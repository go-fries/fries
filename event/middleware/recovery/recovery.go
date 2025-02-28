package recovery

import (
	"context"
	"log"

	"github.com/go-fries/fries/event/v3"
)

type HandlerFunc func(ctx context.Context, event any, recovery any)

var DefaultHandler = func(_ context.Context, event any, recovery any) {
	log.Printf("panic recovery event: %v, recovery: %v", event, recovery)
}

type options struct {
	handler HandlerFunc
}

type Option func(*options)

func WithHandler(h HandlerFunc) Option {
	return func(o *options) {
		if h != nil {
			o.handler = h
		}
	}
}

func New(opts ...Option) event.Middleware {
	o := &options{
		handler: DefaultHandler,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(next event.Handler) event.Handler {
		return func(ctx context.Context, event any) error {
			defer func() {
				if r := recover(); r != nil {
					o.handler(ctx, event, r)
				}
			}()
			return next(ctx, event)
		}
	}
}
