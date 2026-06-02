package otel

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-fries/fries/hyperf/jet/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

func TestTracing(t *testing.T) {
	imsb := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(imsb))
	middleware := New(
		WithTracerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "tracing")),
	)

	tests := []struct {
		name string
		err  error
	}{
		{"success", nil},
		{"error", assert.AnError},
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
			_, err := handler(t.Context(), "service", "method", nil)
			require.Equal(t, tt.err, err)

			spans := imsb.GetSpans()
			require.Len(t, spans, 1)
			span := spans[0]

			assert.Equal(t, trace.SpanKindClient, span.SpanKind)
			assert.Equal(t, "service/method", span.Name)
			assert.Equal(t, scopeName, span.InstrumentationScope.Name)
			assert.Equal(t, "1.2.3", span.InstrumentationScope.Version)
			assert.Equal(t, "https://example.com/schema", span.InstrumentationScope.SchemaURL)
			scopeValue, ok := span.InstrumentationScope.Attributes.Value(attribute.Key("component"))
			require.True(t, ok)
			assert.Equal(t, "tracing", scopeValue.AsString())

			assert.Contains(t, span.Attributes, otelsemconv.RPCMethod("service/method"))

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

func TestTracingWithJetClientContext(t *testing.T) {
	imsb := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(imsb))
	middleware := New(WithTracerProvider(provider))

	transporter, err := jet.NewHTTPTransporter(
		jet.WithHTTPTransporterAddr("https://api.example.com:9443/rpc"),
		jet.WithHTTPTransporterClient(http.DefaultClient),
	)
	require.NoError(t, err)
	client, err := jet.NewClient(
		jet.WithService("service"),
		jet.WithTransporter(transporter),
	)
	require.NoError(t, err)
	ctx := jet.ContextWithClient(t.Context(), client)

	handler := middleware(func(_ context.Context, service, method string, request any) (response any, err error) {
		return nil, nil
	})
	_, err = handler(ctx, "service", "method", nil)
	require.NoError(t, err)

	spans := imsb.GetSpans()
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Equal(t, "/service/method", span.Name)
	assert.Contains(t, span.Attributes, otelsemconv.RPCMethod("/service/method"))
	assert.Contains(t, span.Attributes, otelsemconv.RPCSystemNameJSONRPC)
	assert.Contains(t, span.Attributes, otelsemconv.JSONRPCProtocolVersion(jet.JSONRPCVersion))
	assert.Contains(t, span.Attributes, otelsemconv.HTTPRequestMethodPost)
	assert.Contains(t, span.Attributes, otelsemconv.URLFull("https://api.example.com:9443/rpc"))
	assert.Contains(t, span.Attributes, otelsemconv.ServerAddress("api.example.com"))
	assert.Contains(t, span.Attributes, otelsemconv.ServerPort(9443))
}

func TestTracingWithSemanticErrorAttributes(t *testing.T) {
	imsb := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(imsb))
	middleware := New(WithTracerProvider(provider))
	rpcErr := &jet.RPCResponseError{
		Code:    -32603,
		Message: "internal error",
		Err:     errors.New("boom"),
	}

	handler := middleware(func(_ context.Context, service, method string, request any) (response any, err error) {
		return nil, rpcErr
	})
	_, err := handler(t.Context(), "service", "method", nil)
	require.ErrorIs(t, err, rpcErr)

	spans := imsb.GetSpans()
	require.Len(t, spans, 1)
	span := spans[0]

	assert.Contains(t, span.Attributes, otelsemconv.RPCResponseStatusCode("-32603"))
	assert.Contains(t, span.Attributes, otelsemconv.ErrorTypeKey.String("-32603"))
}
