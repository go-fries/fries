package tracing

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewConfigDefaults(t *testing.T) {
	cfg := newConfig()

	if cfg.tracerProvider == nil {
		t.Fatal("expected tracer provider")
	}
	if cfg.propagator == nil {
		t.Fatal("expected propagator")
	}
	if cfg.version != Version() {
		t.Fatalf("expected version %q, got %q", Version(), cfg.version)
	}
}

func TestNewConfigFallsBackToGlobalTracerProvider(t *testing.T) {
	cfg := newConfig(WithTracerProvider(nil))

	if cfg.tracerProvider == nil {
		t.Fatal("expected tracer provider")
	}
}

func TestConfigNewTracerUsesScopeOptions(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	cfg := newConfig(
		WithTracerProvider(provider),
		WithVersion("1.2.3"),
		WithSchemaURL("https://example.com/schema"),
		WithAttributes(attribute.String("component", "tracing")),
		WithAttributes(attribute.String("layer", "kratos")),
	)

	tracer := cfg.newTracer(scopeName)
	_, span := tracer.Start(t.Context(), "operation")
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected 1 ended span, got %d", len(ended))
	}

	scope := ended[0].InstrumentationScope()
	if scope.Name != scopeName {
		t.Fatalf("expected scope name %q, got %q", scopeName, scope.Name)
	}
	if scope.Version != "1.2.3" {
		t.Fatalf("expected scope version %q, got %q", "1.2.3", scope.Version)
	}
	if scope.SchemaURL != "https://example.com/schema" {
		t.Fatalf("expected schema URL %q, got %q", "https://example.com/schema", scope.SchemaURL)
	}

	got, ok := scope.Attributes.Value(attribute.Key("component"))
	if !ok || got.AsString() != "tracing" {
		t.Fatalf("expected component attribute %q, got %q", "tracing", got.AsString())
	}
	got, ok = scope.Attributes.Value(attribute.Key("layer"))
	if !ok || got.AsString() != "kratos" {
		t.Fatalf("expected layer attribute %q, got %q", "kratos", got.AsString())
	}
}
