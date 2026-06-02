package otlp

import (
	"context"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

// Signal identifies an OpenTelemetry signal configured by [Client].
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

const (
	defaultTraceBatchTimeout  = 10 * time.Second
	defaultTraceExportTimeout = 10 * time.Second
	defaultMetricInterval     = 15 * time.Second
	defaultLogExportInterval  = 10 * time.Second
	defaultLogExportTimeout   = 10 * time.Second
)

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
	traceSampler       sdktrace.Sampler
	traceBatchTimeout  time.Duration
	traceExportTimeout time.Duration

	// signal options
	signals Signal

	// metric options
	metricInterval time.Duration

	// log options
	logExportInterval time.Duration
	logExportTimeout  time.Duration

	// batch options
	batchQueueSize int

	// hooks
	hooks []Hook
}

// Option configures a [Client].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithResource sets the OpenTelemetry resource used by created SDK providers.
//
// When resource is nil, the client creates a default [sdkresource.Resource].
func WithResource(resource *sdkresource.Resource) Option {
	return optionFunc(func(c *config) {
		c.resource = resource
	})
}

// WithPropagator sets the global text map propagator.
//
// When propagator is nil, the client uses a composite propagator containing
// Trace Context and Baggage.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return optionFunc(func(c *config) {
		c.propagator = propagator
	})
}

// WithTracerProvider sets the global trace provider used by [Client.Configure].
//
// When provider is set, the client does not create a trace exporter or SDK trace
// provider for the trace signal.
func WithTracerProvider(provider trace.TracerProvider) Option {
	return optionFunc(func(c *config) {
		c.tracerProvider = provider
	})
}

// WithMeterProvider sets the global meter provider used by [Client.Configure].
//
// When provider is set, the client does not create a metric exporter or SDK
// meter provider for the metric signal.
func WithMeterProvider(provider metric.MeterProvider) Option {
	return optionFunc(func(c *config) {
		c.meterProvider = provider
	})
}

// WithLoggerProvider sets the global logger provider used by [Client.Configure].
//
// When provider is set, the client does not create a log exporter or SDK logger
// provider for the log signal.
func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(c *config) {
		c.loggerProvider = provider
	})
}

// WithSignals sets which OpenTelemetry signals the client configures.
//
// The selected signals override the constructor default. [NewClient] enables all
// signals by default, while the single-signal constructors enable only their own
// signal.
func WithSignals(signals ...Signal) Option {
	return optionFunc(func(c *config) {
		c.signals = combineSignals(signals...)
	})
}

// WithServiceName adds the OpenTelemetry service.name resource attribute.
//
// The value is used only when the client creates the resource.
func WithServiceName(serviceName string) Option {
	return optionFunc(func(c *config) {
		c.serviceName = serviceName
	})
}

// WithDeploymentEnvironmentName adds the deployment.environment.name resource attribute.
//
// The value is used only when the client creates the resource.
func WithDeploymentEnvironmentName(deploymentEnvironment string) Option {
	return optionFunc(func(c *config) {
		c.deploymentEnvironmentName = deploymentEnvironment
	})
}

// WithAttributes appends resource attributes used when the client creates the resource.
func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.attributes = append(c.attributes, attributes...)
	})
}

// WithTraceSampler sets the sampler used when the client creates an SDK trace provider.
func WithTraceSampler(sampler sdktrace.Sampler) Option {
	return optionFunc(func(c *config) {
		c.traceSampler = sampler
	})
}

// WithBatchQueueSize sets the trace and log batch processor queue size.
//
// Non-positive values are ignored.
func WithBatchQueueSize(size int) Option {
	return optionFunc(func(c *config) {
		if size > 0 {
			c.batchQueueSize = size
		}
	})
}

// WithTraceBatchTimeout sets the trace batch processor schedule delay.
//
// Non-positive values are ignored.
func WithTraceBatchTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.traceBatchTimeout = timeout
		}
	})
}

// WithTraceExportTimeout sets the trace batch processor export timeout.
//
// Non-positive values are ignored.
func WithTraceExportTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.traceExportTimeout = timeout
		}
	})
}

// WithMetricInterval sets the periodic metric reader collection interval.
//
// Non-positive values are ignored.
func WithMetricInterval(interval time.Duration) Option {
	return optionFunc(func(c *config) {
		if interval > 0 {
			c.metricInterval = interval
		}
	})
}

// WithLogExportInterval sets the log batch processor export interval.
//
// Non-positive values are ignored.
func WithLogExportInterval(interval time.Duration) Option {
	return optionFunc(func(c *config) {
		if interval > 0 {
			c.logExportInterval = interval
		}
	})
}

// WithLogExportTimeout sets the log batch processor export timeout.
//
// Non-positive values are ignored.
func WithLogExportTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.logExportTimeout = timeout
		}
	})
}

// WithHooks adds hooks that run after [Client.Configure] finishes provider setup.
func WithHooks(hooks ...Hook) Option {
	return optionFunc(func(c *config) {
		if len(hooks) > 0 {
			c.hooks = append(c.hooks, hooks...)
		}
	})
}

func newConfig(signals Signal, opts ...Option) *config {
	cfg := &config{
		signals:            signals,
		traceBatchTimeout:  defaultTraceBatchTimeout,
		traceExportTimeout: defaultTraceExportTimeout,
		metricInterval:     defaultMetricInterval,
		logExportInterval:  defaultLogExportInterval,
		logExportTimeout:   defaultLogExportTimeout,
		batchQueueSize:     queueSize(),
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
		attrs = append(attrs, semconv.DeploymentEnvironmentNameKey.String(c.deploymentEnvironmentName))
	}

	return sdkresource.New(
		ctx,
		sdkresource.WithHost(),
		sdkresource.WithTelemetrySDK(),
		sdkresource.WithContainer(),
		sdkresource.WithAttributes(attrs...),
	)
}

func (c *config) newTextMapPropagator() propagation.TextMapPropagator {
	if c.propagator != nil {
		return c.propagator
	}

	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func (c *config) newTracerProvider(ctx context.Context) (trace.TracerProvider, error) {
	if c.tracerProvider != nil {
		return c.tracerProvider, nil
	}
	if c.traceTransport == nil {
		return nil, ErrTraceTransportRequired
	}

	exporter, err := c.traceTransport.GetTraceSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(
			exporter,
			sdktrace.WithMaxQueueSize(c.batchQueueSize),
			sdktrace.WithMaxExportBatchSize(c.batchQueueSize),
			sdktrace.WithBatchTimeout(c.traceBatchTimeout),
			sdktrace.WithExportTimeout(c.traceExportTimeout),
		)),
		sdktrace.WithResource(c.resource),
		sdktrace.WithSampler(c.traceSampler),
	), nil
}

func (c *config) newMeterProvider(ctx context.Context) (metric.MeterProvider, error) {
	if c.meterProvider != nil {
		return c.meterProvider, nil
	}
	if c.metricTransport == nil {
		return nil, ErrMetricTransportRequired
	}

	exporter, err := c.metricTransport.GetMetricExporter(ctx)
	if err != nil {
		return nil, err
	}

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(c.metricInterval)),
		), //nolint:mnd
		sdkmetric.WithResource(c.resource),
	), nil
}

func (c *config) newLoggerProvider(ctx context.Context) (log.LoggerProvider, error) {
	if c.loggerProvider != nil {
		return c.loggerProvider, nil
	}
	if c.logTransport == nil {
		return nil, ErrLogTransportRequired
	}

	exporter, err := c.logTransport.GetLogExporter(ctx)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(
			exporter,
			sdklog.WithMaxQueueSize(c.batchQueueSize),
			sdklog.WithExportMaxBatchSize(c.batchQueueSize),
			sdklog.WithExportInterval(c.logExportInterval),
			sdklog.WithExportTimeout(c.logExportTimeout),
		)),
		sdklog.WithResource(c.resource),
	), nil
}

func combineSignals(signals ...Signal) Signal {
	var enabled Signal
	for _, signal := range signals {
		enabled |= signal & allSignals
	}
	return enabled
}

func queueSize() int {
	const _min = 1000  //nolint:mnd
	const _max = 16000 //nolint:mnd

	n := (runtime.GOMAXPROCS(0) / 2) * 1000 //nolint:mnd
	if n < _min {
		return _min
	}
	if n > _max {
		return _max
	}
	return n
}
