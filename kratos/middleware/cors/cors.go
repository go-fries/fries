package cors

import (
	"context"
	"regexp"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type options struct {
	paths          []string
	allowedMethods []string
	allowedOrigins []string
	allowedHeaders []string
}

func newOptions() *options {
	return &options{
		paths:          []string{"*"},
		allowedMethods: []string{"*"},
		allowedOrigins: []string{"*"},
		allowedHeaders: []string{"*"},
	}
}

type Option func(*options)

func Path(paths ...string) Option {
	return func(o *options) {
		o.paths = paths
	}
}

func AppendPath(paths ...string) Option {
	return func(o *options) {
		o.paths = append(o.paths, paths...)
	}
}

func AllowedMethods(methods ...string) Option {
	return func(o *options) {
		o.allowedMethods = methods
	}
}

func AppendAllowedMethods(methods ...string) Option {
	return func(o *options) {
		o.allowedMethods = append(o.allowedMethods, methods...)
	}
}

func AllowedHeaders(headers ...string) Option {
	return func(o *options) {
		o.allowedHeaders = headers
	}
}

func AppendAllowedHeaders(headers ...string) Option {
	return func(o *options) {
		o.allowedHeaders = append(o.allowedHeaders, headers...)
	}
}

func AllowedOrigins(origins ...string) Option {
	return func(o *options) {
		o.allowedOrigins = origins
	}
}

func AppendAllowedOrigins(origins ...string) Option {
	return func(o *options) {
		o.allowedOrigins = append(o.allowedOrigins, origins...)
	}
}

func Cors(opts ...Option) middleware.Middleware {
	op := newOptions()

	for _, o := range opts {
		o(op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok || tr.Kind() != transport.KindHTTP {
				return handler(ctx, req)
			}

			request, ok := http.RequestFromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			for _, path := range op.paths {
				if !isPath(path, request.RequestURI) {
					continue
				}

				tr.ReplyHeader().Add("Access-Control-Allow-Credentials", "true")
				tr.ReplyHeader().Add("Access-Control-Allow-Methods", strings.Join(op.allowedMethods, ", "))
				tr.ReplyHeader().Add("Access-Control-Allow-Headers", strings.Join(op.allowedHeaders, ", "))
				tr.ReplyHeader().Add("Access-Control-Allow-Origin", strings.Join(op.allowedOrigins, ", "))

				break
			}

			return handler(ctx, req)
		}
	}
}

// isPath checks if the path matches the pattern
func isPath(pattern, value string) bool {
	if pattern == "*" || pattern == value {
		return true
	}

	pattern = strings.TrimLeft(pattern, "/")
	value = strings.TrimLeft(value, "/")

	pattern = strings.ReplaceAll(pattern, "*", ".*")

	match, err := regexp.MatchString(pattern, value)

	return err == nil && match
}
