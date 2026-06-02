package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	tracerProvider trace.TracerProvider
	version        string
	schemaURL      string
	attributes     []attribute.KeyValue
}

// Option configures tracing middleware.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithTracerProvider sets the [trace.TracerProvider] used to create the
// underlying [trace.Tracer]. A nil provider falls back to the global provider.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		if provider != nil {
			c.tracerProvider = provider
		}
	})
}

// WithVersion sets the instrumentation scope version reported to
// OpenTelemetry.
func WithVersion(version string) Option {
	return optionFunc(func(c *config) {
		if version != "" {
			c.version = version
		}
	})
}

// WithSchemaURL sets the schema URL reported to OpenTelemetry for spans emitted
// by the instrumentation scope.
func WithSchemaURL(schemaURL string) Option {
	return optionFunc(func(c *config) {
		if schemaURL != "" {
			c.schemaURL = schemaURL
		}
	})
}

// WithAttributes adds instrumentation scope [attribute.KeyValue] attributes
// reported to OpenTelemetry.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.attributes = append(c.attributes, attributes...)
	})
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		version: Version(),
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	if cfg.tracerProvider == nil {
		cfg.tracerProvider = otel.GetTracerProvider()
	}
	return cfg
}

func (c *config) newTracer(name string) trace.Tracer {
	opts := []trace.TracerOption{
		trace.WithInstrumentationVersion(c.version),
	}
	if c.schemaURL != "" {
		opts = append(opts, trace.WithSchemaURL(c.schemaURL))
	}
	if len(c.attributes) > 0 {
		opts = append(opts, trace.WithInstrumentationAttributes(c.attributes...))
	}

	return c.tracerProvider.Tracer(name, opts...)
}
