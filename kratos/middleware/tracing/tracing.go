package tracing

import (
	"context"

	"github.com/go-fries/fries/kratos/middleware/tracing/v3/internal/semconv"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace"
)

var spanAttributeBuilder = semconv.NewBuilder(serviceHeader)

// Server returns a new server middleware for OpenTelemetry.
func Server(opts ...Option) middleware.Middleware {
	tracer := newTracer(trace.SpanKindServer, opts...)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				var span trace.Span
				ctx, span = tracer.start(ctx, tr.Operation(), tr.RequestHeader())
				span.SetAttributes(spanAttributeBuilder.Server(ctx, req)...)
				defer func() { tracer.end(ctx, span, reply, err) }()
			}
			return handler(ctx, req)
		}
	}
}

// Client returns a new client middleware for OpenTelemetry.
func Client(opts ...Option) middleware.Middleware {
	tracer := newTracer(trace.SpanKindClient, opts...)
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				var span trace.Span
				ctx, span = tracer.start(ctx, tr.Operation(), tr.RequestHeader())
				span.SetAttributes(spanAttributeBuilder.Client(ctx, req)...)
				defer func() { tracer.end(ctx, span, reply, err) }()
			}
			return handler(ctx, req)
		}
	}
}

// TraceID returns a log valuer for the current span trace ID.
func TraceID() log.Valuer {
	return func(ctx context.Context) any {
		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			return span.TraceID().String()
		}
		return ""
	}
}

// SpanID returns a log valuer for the current span ID.
func SpanID() log.Valuer {
	return func(ctx context.Context) any {
		if span := trace.SpanContextFromContext(ctx); span.HasSpanID() {
			return span.SpanID().String()
		}
		return ""
	}
}
