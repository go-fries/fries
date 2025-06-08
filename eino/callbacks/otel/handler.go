package otel

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/schema"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "github.com/go-fries/fries/eino/callbacks/otel/v3"

type Handler struct {
	tracer trace.Tracer
}

var _ callbacks.Handler = (*Handler)(nil)

func NewHandler(opts ...Option) *Handler {
	o := newOptions(opts...)

	// tracer
	tracer := o.tp.Tracer(scopeName,
		trace.WithInstrumentationVersion(Version()),
	)

	return &Handler{
		tracer: tracer,
	}
}

func (h *Handler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}
	ctx, span := h.tracer.Start(ctx, getName(info),
		trace.WithSpanKind(trace.SpanKindInternal),
	)

	// span with attributes
	spanWithRunInfo(span, info)
	spanWithModelCallbackInput(span, input)

	return withOTelState(ctx, &state{
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
	defer state.spanEnd()

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
	defer state.spanRecordErr(err)

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

// getName returns the name of the component based on the RunInfo.
func getName(info *callbacks.RunInfo) string {
	if len(info.Name) != 0 {
		return info.Name
	}
	return info.Type + string(info.Component)
}
