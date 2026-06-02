package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := newConfig()

	assert.NotNil(t, cfg.tracerProvider)
	assert.Equal(t, Version(), cfg.version)
	assert.Empty(t, cfg.schemaURL)
	assert.Empty(t, cfg.attributes)
}

func TestNewConfigFallsBackToGlobalTracerProvider(t *testing.T) {
	cfg := newConfig(WithTracerProvider(nil))

	assert.NotNil(t, cfg.tracerProvider)
}

func TestWithEmptyVersionAndSchemaURLKeepDefaults(t *testing.T) {
	cfg := newConfig(WithVersion(""), WithSchemaURL(""))

	assert.Equal(t, Version(), cfg.version)
	assert.Empty(t, cfg.schemaURL)
}

func TestConfigOptions(t *testing.T) {
	cfg := newConfig(
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "tracing")),
		WithAttributes(attribute.String("layer", "hyperf")),
	)

	assert.Equal(t, "1.2.3", cfg.version)
	assert.Equal(t, "https://example.com/schema", cfg.schemaURL)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("component", "tracing"),
		attribute.String("layer", "hyperf"),
	}, cfg.attributes)
}

func TestConfigNewTracerUsesScopeOptions(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	cfg := newConfig(
		WithTracerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "tracing")),
		WithAttributes(attribute.String("layer", "hyperf")),
	)

	tracer := cfg.newTracer(scopeName)
	_, span := tracer.Start(t.Context(), "operation")
	span.End()

	ended := recorder.Ended()
	require.Len(t, ended, 1)

	scope := ended[0].InstrumentationScope()
	assert.Equal(t, scopeName, scope.Name)
	assert.Equal(t, "1.2.3", scope.Version)
	assert.Equal(t, "https://example.com/schema", scope.SchemaURL)

	got, ok := scope.Attributes.Value(attribute.Key("component"))
	require.True(t, ok)
	assert.Equal(t, "tracing", got.AsString())
	got, ok = scope.Attributes.Value(attribute.Key("layer"))
	require.True(t, ok)
	assert.Equal(t, "hyperf", got.AsString())
}
