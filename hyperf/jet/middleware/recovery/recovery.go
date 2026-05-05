package recovery

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/hyperf/jet/v3"
)

var DefaultHandler = func(_ context.Context, service, method string, request, err any) error {
	return fmt.Errorf("service: %s, method: %s, request: %v, error: %v", service, method, request, err)
}

type HandlerFunc func(ctx context.Context, service, method string, request, err any) error

type options struct {
	handler HandlerFunc
}

type Option func(*options)

func Handler(h HandlerFunc) Option {
	return func(o *options) {
		o.handler = h
	}
}

func New(opts ...Option) jet.Middleware {
	return func(next jet.Handler) jet.Handler {
		o := options{
			handler: DefaultHandler,
		}
		for _, opt := range opts {
			opt(&o)
		}

		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			defer func() {
				if rerr := recover(); rerr != nil {
					err = o.handler(ctx, service, method, request, rerr)
				}
			}()
			return next(ctx, service, method, request)
		}
	}
}
