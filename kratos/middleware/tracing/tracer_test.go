package tracing

import (
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	tracingpb "github.com/go-fries/fries/kratos/middleware/tracing/v3/internal/proto/tracing/v1"
)

func TestNewTracer(t *testing.T) {
	tracer := newTracer(trace.SpanKindClient, WithTracerProvider(noop.NewTracerProvider()))

	if tracer.kind != trace.SpanKindClient {
		t.Errorf("The tracer kind must be equal to trace.SpanKindClient, %v given.", tracer.kind)
	}

	defer func() {
		if recover() == nil {
			t.Error("newTracer with an invalid SpanKindMustCrash must panic")
		}
	}()
	_ = newTracer(666, WithTracerProvider(noop.NewTracerProvider()))
}

func TestTracer_End(t *testing.T) {
	tracer := newTracer(trace.SpanKindClient, WithTracerProvider(noop.NewTracerProvider()))
	ctx, span := noop.NewTracerProvider().Tracer("noop").Start(t.Context(), "noopSpan")

	// Handle with error case
	tracer.end(ctx, span, nil, errors.New("dummy error"))

	// Handle without error case
	tracer.end(ctx, span, nil, nil)

	m := &tracingpb.HelloRequest{}

	// Handle the trace KindServer
	tracer = newTracer(trace.SpanKindServer, WithTracerProvider(noop.NewTracerProvider()))
	tracer.end(ctx, span, m, nil)
	tracer = newTracer(trace.SpanKindClient, WithTracerProvider(noop.NewTracerProvider()))
	tracer.end(ctx, span, m, nil)
}
