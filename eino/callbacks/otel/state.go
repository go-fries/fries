package otel

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
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

func spanWithRunInfo(span trace.Span, info *callbacks.RunInfo) {
	span.SetAttributes(
		attribute.String("eino.runinfo.type", info.Type),
		attribute.String("eino.runinfo.name", info.Name),
		attribute.String("eino.runinfo.component", string(info.Component)),
	)
}

func spanWithModelCallbackInput(span trace.Span, input callbacks.CallbackInput) {
	callbackInput := model.ConvCallbackInput(input)
	if callbackInput == nil {
		return
	}

	if config := callbackInput.Config; config != nil {
		span.SetAttributes(
			semconv.GenAIRequestModel(config.Model),
			semconv.GenAIRequestMaxTokens(config.MaxTokens),
			semconv.GenAIRequestTemperature(float64(config.Temperature)),
			semconv.GenAIRequestTopP(float64(config.TopP)),
		)
	}

	if len(callbackInput.Messages) > 0 {
		for i, msg := range callbackInput.Messages {
			span.AddEvent(
				fmt.Sprintf("input.message.%d", i+1),
				trace.WithAttributes(
					attribute.String("gen_ai.request.role", string(msg.Role)),
					attribute.String("gen_ai.request.content", msg.Content),
					// Add more attributes as needed
				),
			)
		}
	}

	if extra := callbackInput.Extra; len(extra) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(extra))
		for k, v := range extra {
			attrs = append(attrs, attribute.String("callback.input.extra."+k, fmt.Sprintf("%v", v)))
		}
		span.SetAttributes(attrs...)
	}
}
