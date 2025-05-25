package otel

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
)

// otelStateKey is a context key for storing OpenTelemetry state.
type otelStateKey struct{}

// otelState holds the OpenTelemetry span for the current context.
type otelState struct {
	span trace.Span
}

// withOTelState adds the OpenTelemetry state to the context.
func withOTelState(ctx context.Context, state *otelState) context.Context {
	return context.WithValue(ctx, otelStateKey{}, state)
}

// fromOTelState retrieves the OpenTelemetry state from the context.
func fromOTelState(ctx context.Context) (*otelState, bool) {
	state, ok := ctx.Value(otelStateKey{}).(*otelState)
	return state, ok
}

// getName returns the name of the component based on the RunInfo.
func getName(info *callbacks.RunInfo) string {
	if len(info.Name) != 0 {
		return info.Name
	}
	return info.Type + string(info.Component)
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
