package otel

import (
	"fmt"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"go.opentelemetry.io/otel/trace"
)

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
				fmt.Sprintf("gen_ai.request.message.%d", i+1),
				trace.WithAttributes(
					attribute.String("gen_ai.request.message.role", string(msg.Role)),
					attribute.String("gen_ai.request.message.content", msg.Content),
					// Add more attributes as needed, ex: tools
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

func spanWithModelCallbackOutput(span trace.Span, output callbacks.CallbackOutput) {
	callbackOutput := model.ConvCallbackOutput(output)
	if callbackOutput == nil {
		return
	}

	if config := callbackOutput.Config; config != nil {
		span.SetAttributes(
			semconv.GenAIResponseModel(config.Model),
		)
	}

	if callbackOutput.Message != nil {
		span.AddEvent(
			"gen_ai.response.message",
			trace.WithAttributes(
				attribute.String("gen_ai.response.message.role", string(callbackOutput.Message.Role)),
				attribute.String("gen_ai.response.message.content", callbackOutput.Message.Content),
				// Add more attributes as needed, ex: tools
			),
		)
	}

	if extra := callbackOutput.Extra; len(extra) > 0 {
		attrs := make([]attribute.KeyValue, 0, len(extra))
		for k, v := range extra {
			attrs = append(attrs, attribute.String("callback.output.extra."+k, fmt.Sprintf("%v", v)))
		}
		span.SetAttributes(attrs...)
	}
}
