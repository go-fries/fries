package logger

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-fries/fries/hyperf/jet/v4"
)

// New creates logger middleware.
func New(opts ...Option) jet.Middleware {
	cfg := newConfig(opts...)
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			defer func(starting time.Time) {
				level := slog.LevelInfo
				if err != nil {
					level = slog.LevelError
				}

				cfg.logger.LogAttrs(
					ctx,
					level,
					"jet request",
					slog.String("kind", "jet"),
					slog.String("service", service),
					slog.String("method", method),
					slog.Any("request", request),
					slog.Any("response", response),
					slog.Any("error", err),
					slog.Duration("latency", time.Since(starting)),
				)
			}(time.Now())
			return next(ctx, service, method, request)
		}
	}
}
