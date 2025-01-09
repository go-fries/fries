package protovalidate

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"google.golang.org/protobuf/proto"
)

var defaultHandler = func(_ context.Context, _ any, err error) (any, error) {
	return nil, errors.BadRequest("VALIDATE_ERROR", fmt.Sprintf("invalid request: %v", err))
}

type options struct {
	validator *protovalidate.Validator
	handler   func(ctx context.Context, req any, err error) (any, error)
}

type Option func(*options)

func Validator(validator *protovalidate.Validator) Option {
	return func(opts *options) {
		opts.validator = validator
	}
}

func Handler(handler func(ctx context.Context, req any, err error) (any, error)) Option {
	return func(opts *options) {
		opts.handler = handler
	}
}

func Server(opts ...Option) middleware.Middleware {
	o := &options{
		handler: defaultHandler,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if o.validator != nil {
				if msg, ok := req.(proto.Message); ok {
					if err := o.validator.Validate(msg); err != nil {
						return o.handler(ctx, req, err)
					}
				}
			}
			return handler(ctx, req)
		}
	}
}
