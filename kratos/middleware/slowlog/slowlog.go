package slowlog

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// DefaultThreshold Threshold is the slow query threshold, default is 2000ms
const defaultThreshold = 2 * time.Second

type options struct {
	Threshold time.Duration
}

type Option func(*options)

func WithThreshold(threshold time.Duration) Option {
	return func(c *options) {
		c.Threshold = threshold
	}
}

// Redacter defines how to log an object
type Redacter interface {
	Redact() string
}

// Server returns a middleware that logs slow requests from the server.
func Server(logger log.Logger, opts ...Option) middleware.Middleware {
	o := &options{
		Threshold: defaultThreshold,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()

			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			reply, err = handler(ctx, req)
			if se := errors.FromError(err); se != nil {
				code = se.Code
				reason = se.Reason
			}

			duration := time.Since(startTime)

			if duration <= o.Threshold {
				return
			}

			level, stack := extractError(err)
			log.NewHelper(log.WithContext(ctx, logger)).Log(level,
				"kind", "server",
				"type", "slowlog",
				"component", kind,
				"operation", operation,
				"args", extractArgs(req),
				"code", code,
				"reason", reason,
				"stack", stack,
				"latency", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

// Client returns a middleware that logs slow requests from the client.
func Client(logger log.Logger, opts ...Option) middleware.Middleware {
	o := &options{
		Threshold: defaultThreshold,
	}
	for _, opt := range opts {
		opt(o)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			reply, err = handler(ctx, req)
			if se := errors.FromError(err); se != nil {
				code = se.Code
				reason = se.Reason
			}

			duration := time.Since(startTime)
			if duration <= o.Threshold {
				return
			}

			level, stack := extractError(err)
			log.NewHelper(log.WithContext(ctx, logger)).Log(level,
				"kind", "client",
				"type", "slowlog",
				"component", kind,
				"operation", operation,
				"args", extractArgs(req),
				"code", code,
				"reason", reason,
				"stack", stack,
				"latency", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

// extractError returns the string of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}

// extractArgs returns the string of the req
func extractArgs(req any) string {
	if redacter, ok := req.(Redacter); ok {
		return redacter.Redact()
	}
	if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}
