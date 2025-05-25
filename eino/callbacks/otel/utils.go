package otel

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"go.opentelemetry.io/otel/trace"
)

type otelStateKey struct{}

type otelState struct {
	span trace.Span
}

func withOTelState(ctx context.Context, state *otelState) context.Context {
	return context.WithValue(ctx, otelStateKey{}, state)
}

func fromOTelState(ctx context.Context) (*otelState, bool) {
	state, ok := ctx.Value(otelStateKey{}).(*otelState)
	return state, ok
}

func getName(info *callbacks.RunInfo) string {
	if len(info.Name) != 0 {
		return info.Name
	}
	return info.Type + string(info.Component)
}
