package tracing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTracing(t *testing.T) {
	imsb := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(imsb))
	middleware := New(WithTracerProvider(provider))

	tests := []struct {
		name string
		err  error
	}{
		{"", nil},
		{"", assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer imsb.Reset()

			handler := middleware(func(_ context.Context, service, method string, request any) (response any, err error) {
				assert.Equal(t, "service", service)
				assert.Equal(t, "method", method)
				assert.Nil(t, request)
				return nil, tt.err
			})
			_, err := handler(context.Background(), "service", "method", nil)
			require.Equal(t, tt.err, err)

			spans := imsb.GetSpans()
			require.Len(t, spans, 1)
			span := spans[0]

			assert.Equal(t, trace.SpanKindClient, span.SpanKind)
			assert.Equal(t, "jet.service/method", span.Name)

			// rpc.service, rpc.method
			assert.Contains(t, span.Attributes, attribute.String("rpc.service", "service"))
			assert.Contains(t, span.Attributes, attribute.String("rpc.method", "method"))

			if tt.err != nil {
				assert.Equal(t, codes.Error, span.Status.Code)
				assert.Equal(t, tt.err.Error(), span.Status.Description)

				// events
				require.Len(t, span.Events, 1)
				event := span.Events[0]
				assert.Equal(t, "exception", event.Name)
				assert.Contains(t, event.Attributes, attribute.String("exception.type", "*errors.errorString"))
				assert.Contains(t, event.Attributes, attribute.String("exception.message", tt.err.Error()))
			}
		})
	}
}
