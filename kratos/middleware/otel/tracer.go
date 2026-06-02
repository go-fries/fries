package otel

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/kratos/middleware/otel/v3/internal/semconv"
	"github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/go-fries/fries/kratos/middleware/otel/v3"

type tracer struct {
	tracer     trace.Tracer
	kind       trace.SpanKind
	propagator propagation.TextMapPropagator
}

func newTracer(kind trace.SpanKind, opts ...Option) *tracer {
	cfg := newConfig(opts...)

	switch kind {
	case trace.SpanKindClient:
	case trace.SpanKindServer:
	default:
		panic(fmt.Sprintf("unsupported span kind: %v", kind))
	}

	return &tracer{
		tracer:     cfg.newTracer(scopeName),
		kind:       kind,
		propagator: cfg.propagator,
	}
}

func (t *tracer) start(
	ctx context.Context, operation string, carrier propagation.TextMapCarrier,
) (context.Context, trace.Span) {
	if t.kind == trace.SpanKindServer {
		ctx = t.propagator.Extract(ctx, carrier)
	}
	ctx, span := t.tracer.Start(
		ctx,
		operation,
		trace.WithSpanKind(t.kind),
	)
	if t.kind == trace.SpanKindClient {
		t.propagator.Inject(ctx, carrier)
	}
	return ctx, span
}

func (t *tracer) end(_ context.Context, span trace.Span, m any, err error) {
	if err != nil {
		span.RecordError(err)
		if e := errors.FromError(err); e != nil {
			span.SetAttributes(semconv.RPCErrorAttributes(e.Code)...)
		}
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	if t.kind == trace.SpanKindServer {
		span.SetAttributes(semconv.SendMessageSize(m)...)
	} else {
		span.SetAttributes(semconv.RecvMessageSize(m)...)
	}
	span.End()
}
