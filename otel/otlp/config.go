package otlp

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

// Signal identifies an OpenTelemetry signal configured by Client.
type Signal uint8

const (
	// TraceSignal configures the global OpenTelemetry trace provider.
	TraceSignal Signal = 1 << iota
	// MetricSignal configures the global OpenTelemetry meter provider.
	MetricSignal
	// LogSignal configures the global OpenTelemetry logger provider.
	LogSignal
)

const allSignals = TraceSignal | MetricSignal | LogSignal

type config struct {
	// otlp transports
	traceTransport  TraceTransport
	metricTransport MetricTransport
	logTransport    LogTransport

	// core components
	resource       *sdkresource.Resource
	propagator     propagation.TextMapPropagator
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	loggerProvider log.LoggerProvider

	// resource options
	serviceName               string
	deploymentEnvironmentName string
	attributes                []attribute.KeyValue

	// trace options
	traceSampler sdktrace.Sampler

	// signal options
	signals Signal

	// hooks
	hooks []Hook
}

// Option configures the OTLP client.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func WithResource(resource *sdkresource.Resource) Option {
	return optionFunc(func(c *config) {
		c.resource = resource
	})
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		c.propagator = propagator
	})
}

func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		c.tracerProvider = provider
	})
}

func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		c.meterProvider = provider
	})
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(c *config) {
		c.loggerProvider = provider
	})
}

// WithSignals sets which OpenTelemetry signals the client configures.
func WithSignals(signals ...Signal) Option {
	return optionFunc(func(c *config) {
		c.signals = combineSignals(signals...)
	})
}

func WithServiceName(serviceName string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = serviceName
	})
}

func WithDeploymentEnvironmentName(deploymentEnvironment string) Option {
	return optionFunc(func(c *config) {
		c.deploymentEnvironmentName = deploymentEnvironment
	})
}

func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.attributes = append(c.attributes, attributes...)
	})
}

func WithTraceSampler(sampler sdktrace.Sampler) Option {
	return optionFunc(func(c *config) {
		c.traceSampler = sampler
	})
}

func WithHooks(hooks ...Hook) Option {
	return optionFunc(func(c *config) {
		if len(hooks) > 0 {
			c.hooks = append(c.hooks, hooks...)
		}
	})
}

func newConfig(signals Signal, opts ...Option) *config {
	cfg := &config{
		signals: signals,
		hooks:   DefaultHooks(),
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

func (c *config) signalEnabled(signal Signal) bool {
	return c.signals&signal != 0
}

func (c *config) newResource(ctx context.Context) (*sdkresource.Resource, error) {
	if c.resource != nil {
		return c.resource, nil
	}

	attrs := c.attributes
	if c.serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(c.serviceName))
	}
	if c.deploymentEnvironmentName != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(c.deploymentEnvironmentName))
	}

	return sdkresource.New(
		ctx,
		sdkresource.WithHost(),
		sdkresource.WithTelemetrySDK(),
		sdkresource.WithContainer(),
		sdkresource.WithAttributes(attrs...),
	)
}

func combineSignals(signals ...Signal) Signal {
	var enabled Signal
	for _, signal := range signals {
		enabled |= signal & allSignals
	}
	return enabled
}
