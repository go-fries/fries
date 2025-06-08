package otel

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// stateKey is a context key for storing OpenTelemetry state.
type stateKey struct{}

// state holds the OpenTelemetry span for the current context.
type state struct {
	span trace.Span
}

func (s *state) spanRecordErr(err error) {
	if s.span == nil || !s.span.IsRecording() {
		return
	}

	if err != nil {
		s.span.RecordError(err)
		s.span.SetStatus(codes.Error, err.Error())
	}
}

func (s *state) spanEnd() {
	if s.span == nil || !s.span.IsRecording() {
		return
	}

	s.span.End()
}

// withOTelState adds the OpenTelemetry state to the context.
func withOTelState(ctx context.Context, state *state) context.Context {
	return context.WithValue(ctx, stateKey{}, state)
}

// fromOTelState retrieves the OpenTelemetry state from the context.
func fromOTelState(ctx context.Context) (*state, bool) {
	s, ok := ctx.Value(stateKey{}).(*state)
	return s, ok
}
