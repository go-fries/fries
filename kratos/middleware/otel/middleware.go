package otel

import (
	"context"

	"github.com/go-fries/fries/kratos/middleware/otel/v3/internal/semconv"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace"
)

var spanAttributeBuilder = semconv.NewBuilder(serviceHeader)

// Server returns a new [middleware.Middleware] for server-side OpenTelemetry tracing.
func Server(opts ...Option) middleware.Middleware {
	t := newTracer(trace.SpanKindServer, opts...)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				var span trace.Span
				ctx, span = t.start(ctx, tr.Operation(), tr.RequestHeader())
				span.SetAttributes(spanAttributeBuilder.Server(ctx, req)...)
				defer func() { t.end(ctx, span, reply, err) }()
			}
			return handler(ctx, req)
		}
	}
}

// Client returns a new [middleware.Middleware] for client-side OpenTelemetry tracing.
func Client(opts ...Option) middleware.Middleware {
	t := newTracer(trace.SpanKindClient, opts...)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				var span trace.Span
				ctx, span = t.start(ctx, tr.Operation(), tr.RequestHeader())
				span.SetAttributes(spanAttributeBuilder.Client(ctx, req)...)
				defer func() { t.end(ctx, span, reply, err) }()
			}
			return handler(ctx, req)
		}
	}
}
