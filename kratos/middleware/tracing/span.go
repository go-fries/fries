package tracing

import (
	"context"

	"github.com/go-fries/fries/kratos/middleware/tracing/v3/internal/semconv"
	"go.opentelemetry.io/otel/trace"
)

var spanAttributeBuilder = semconv.NewBuilder(serviceHeader)

func setClientSpan(ctx context.Context, span trace.Span, m any) {
	span.SetAttributes(spanAttributeBuilder.Client(ctx, m)...)
}

func setServerSpan(ctx context.Context, span trace.Span, m any) {
	span.SetAttributes(spanAttributeBuilder.Server(ctx, m)...)
}
