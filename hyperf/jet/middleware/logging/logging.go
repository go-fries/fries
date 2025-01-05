package logging

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-fires/fires/v3/hyperf/jet"
)

type options struct {
	logger log.Logger
}

type Option func(*options)

func Logger(logger log.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func New(opts ...Option) jet.Middleware {
	o := options{
		logger: log.DefaultLogger,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			defer func(starting time.Time) {
				level := log.LevelInfo
				if err != nil {
					level = log.LevelError
				}

				_ = log.WithContext(ctx, o.logger).Log(level,
					"kind", "jet",
					"service", service,
					"method", method,
					"request", request,
					"response", response,
					"error", err,
					"latency", time.Since(starting),
				)
			}(time.Now())
			return next(ctx, service, method, request)
		}
	}
}
