package otel

import (
	"context"

	"github.com/go-fries/fries/hyperf/jet/middleware/otel/v3/internal/semconv"
	"github.com/go-fries/fries/hyperf/jet/v3"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/go-fries/fries/hyperf/jet/middleware/otel/v3"

var spanAttributeBuilder = semconv.NewBuilder()

// New returns a Jet middleware that creates OpenTelemetry client spans.
func New(opts ...Option) jet.Middleware {
	cfg := newConfig(opts...)
	tracer := cfg.newTracer(scopeName)
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			ctx, span := tracer.Start(
				ctx, "jet."+service+"/"+method,
				trace.WithSpanKind(trace.SpanKindClient),
			)
			defer span.End()
			span.SetAttributes(spanAttributeBuilder.Client(ctx, service, method)...)

			response, err = next(ctx, service, method, request)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
				span.SetAttributes(semconv.ErrorAttributes(err)...)
			}

			return response, err
		}
	}
}
