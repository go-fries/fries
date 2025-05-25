package otel

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"github.com/go-fries/fries/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/go-fries/fries/eino/callbacks/otel/v3"

type Handler struct {
	tp trace.TracerProvider

	tracer trace.Tracer
}

var _ callbacks.Handler = (*Handler)(nil)

func NewHandler(opts ...Option) *Handler {
	handler := &Handler{
		tp: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt.apply(handler)
	}

	// tracer
	handler.tracer = handler.tp.Tracer(scopeName,
		trace.WithInstrumentationVersion(fries.Version()),
	)

	return handler
}

func (h *Handler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	spanName := getName(info)

	ctx, span := h.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	return withOTelState(ctx, &otelState{
		span: span,
	})
}

func (h *Handler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}

	state, ok := fromOTelState(ctx)
	if !ok {
		return ctx
	}

	if state.span == nil || !state.span.IsRecording() {
		return ctx
	}

	defer state.span.End()

	return ctx
}

func (h *Handler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		return ctx
	}

	state, ok := fromOTelState(ctx)
	if !ok {
		return ctx
	}

	if state.span == nil || !state.span.IsRecording() {
		return ctx
	}

	if err != nil {
		state.span.RecordError(err)
		state.span.SetStatus(codes.Error, err.Error())
	}

	return ctx
}

func (h *Handler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	// TODO implement me
	return ctx
}

func (h *Handler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	// TODO implement me
	return ctx
}
