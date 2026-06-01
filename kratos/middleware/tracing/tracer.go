package tracing

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

const scopeName = "github.com/go-fries/fries/kratos/middleware/tracing/v3"

// Tracer starts and ends OpenTelemetry spans for Kratos middleware.
type Tracer struct {
	tracer     trace.Tracer
	kind       trace.SpanKind
	propagator propagation.TextMapPropagator
}

// NewTracer creates a [Tracer] for the given span kind.
func NewTracer(kind trace.SpanKind, opts ...Option) *Tracer {
	cfg := newConfig(opts...)

	switch kind {
	case trace.SpanKindClient:
	case trace.SpanKindServer:
	default:
		panic(fmt.Sprintf("unsupported span kind: %v", kind))
	}

	return &Tracer{
		tracer:     cfg.newTracer(scopeName),
		kind:       kind,
		propagator: cfg.propagator,
	}
}

// Start starts a tracing span and propagates context for the configured span
// kind.
func (t *Tracer) Start(
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

// End records the result on span and ends it.
func (t *Tracer) End(_ context.Context, span trace.Span, m any, err error) {
	if err != nil {
		span.RecordError(err)
		if e := errors.FromError(err); e != nil {
			span.SetAttributes(attribute.Key("rpc.status_code").Int64(int64(e.Code)))
		}
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}

	if p, ok := m.(proto.Message); ok {
		if t.kind == trace.SpanKindServer {
			span.SetAttributes(attribute.Key("send_msg.size").Int(proto.Size(p)))
		} else {
			span.SetAttributes(attribute.Key("recv_msg.size").Int(proto.Size(p)))
		}
	}
	span.End()
}
