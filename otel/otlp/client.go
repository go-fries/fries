package otlp

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrTransportRequired       = errors.New("otlp transport is required")
	ErrTraceTransportRequired  = errors.New("otlp trace transport is required")
	ErrMetricTransportRequired = errors.New("otlp metric transport is required")
	ErrLogTransportRequired    = errors.New("otlp log transport is required")
	ErrClientConfigured        = errors.New("otlp client has already been configured")
	ErrClientShutdown          = errors.New("otlp client has been shut down")
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

type Client struct {
	mu sync.Mutex

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
	traceSampler sdktrace.Sampler // default is always on

	// signal options
	signals Signal

	// hooks
	hooks []Hook

	configured bool
	shutdown   bool
}

type Option func(*Client)

func WithResource(resource *sdkresource.Resource) Option {
	return func(c *Client) {
		c.resource = resource
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(c *Client) {
		c.propagator = propagator
	}
}

func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(c *Client) {
		c.tracerProvider = provider
	}
}

func WithMeterProvider(provider metric.MeterProvider) Option {
	return func(c *Client) {
		c.meterProvider = provider
	}
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return func(c *Client) {
		c.loggerProvider = provider
	}
}

// WithSignals sets which OpenTelemetry signals the client configures.
func WithSignals(signals ...Signal) Option {
	return func(c *Client) {
		c.signals = combineSignals(signals...)
	}
}

func WithServiceName(serviceName string) Option {
	return func(c *Client) {
		c.serviceName = serviceName
	}
}

func WithDeploymentEnvironmentName(deploymentEnvironment string) Option {
	return func(c *Client) {
		c.deploymentEnvironmentName = deploymentEnvironment
	}
}

func WithAttributes(attributes ...attribute.KeyValue) Option {
	return func(c *Client) {
		c.attributes = append(c.attributes, attributes...)
	}
}

func WithTraceSampler(sampler sdktrace.Sampler) Option {
	return func(c *Client) {
		c.traceSampler = sampler
	}
}

func WithHooks(hooks ...Hook) Option {
	return func(c *Client) {
		if len(hooks) > 0 {
			c.hooks = append(c.hooks, hooks...)
		}
	}
}

// NewClient creates a Client configured with a transport for all signals.
func NewClient(transport Transport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrTransportRequired
	}

	c := &Client{
		traceTransport:  transport,
		metricTransport: transport,
		logTransport:    transport,
		signals:         allSignals,
		hooks:           DefaultHooks(),
	}
	c.apply(opts...)
	return c, nil
}

// NewTraceClient creates a Client configured with a trace transport.
func NewTraceClient(transport TraceTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrTraceTransportRequired
	}

	c := newClient(TraceSignal)
	c.traceTransport = transport
	c.apply(opts...)
	return c, nil
}

// NewMetricClient creates a Client configured with a metric transport.
func NewMetricClient(transport MetricTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrMetricTransportRequired
	}

	c := newClient(MetricSignal)
	c.metricTransport = transport
	c.apply(opts...)
	return c, nil
}

// NewLogClient creates a Client configured with a log transport.
func NewLogClient(transport LogTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrLogTransportRequired
	}

	c := newClient(LogSignal)
	c.logTransport = transport
	c.apply(opts...)
	return c, nil
}

func newClient(signals Signal) *Client {
	return &Client{
		signals: signals,
		hooks:   DefaultHooks(),
	}
}

func (c *Client) apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

func (c *Client) Configure(ctx context.Context) error {
	c.mu.Lock()
	switch {
	case c.shutdown:
		c.mu.Unlock()
		return ErrClientShutdown
	case c.configured:
		c.mu.Unlock()
		return ErrClientConfigured
	}
	c.mu.Unlock()

	// resource
	if err := c.configureResource(ctx); err != nil {
		return err
	}

	// propagator
	c.configureTextMapPropagator()

	if c.signalEnabled(TraceSignal) {
		if err := c.configureTraceProvider(ctx); err != nil {
			return err
		}
	}

	if c.signalEnabled(MetricSignal) {
		if err := c.configureMeterProvider(ctx); err != nil {
			return err
		}
	}

	if c.signalEnabled(LogSignal) {
		if err := c.configureLoggerProvider(ctx); err != nil {
			return err
		}
	}

	// run configured hooks
	if err := c.runConfiguredHooks(ctx); err != nil {
		return err
	}

	c.mu.Lock()
	if c.shutdown {
		c.mu.Unlock()
		return ErrClientShutdown
	}
	c.configured = true
	c.mu.Unlock()

	kratoslog.Info("OTLP client configured")

	return nil
}

func (c *Client) configureResource(ctx context.Context) error {
	if c.resource != nil {
		return nil
	}

	attrs := c.attributes

	if c.serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(c.serviceName))
	}

	if c.deploymentEnvironmentName != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentName(c.deploymentEnvironmentName))
	}

	res, err := sdkresource.New(
		ctx,
		sdkresource.WithHost(),
		sdkresource.WithTelemetrySDK(),
		sdkresource.WithContainer(),
		sdkresource.WithAttributes(attrs...),
	)
	if err != nil {
		return err
	}

	c.resource = res

	return nil
}

func (c *Client) configureTextMapPropagator() {
	if c.propagator != nil {
		otel.SetTextMapPropagator(c.propagator)
		return
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

func (c *Client) configureTraceProvider(ctx context.Context) error {
	if c.tracerProvider != nil {
		otel.SetTracerProvider(c.tracerProvider)
		return nil
	}

	if c.traceTransport == nil {
		return ErrTraceTransportRequired
	}

	exporter, err := c.traceTransport.GetTraceSpanExporter(ctx)
	if err != nil {
		return err
	}

	queueSize := queueSize()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(
			exporter,
			sdktrace.WithMaxQueueSize(queueSize),
			sdktrace.WithMaxExportBatchSize(queueSize),
			sdktrace.WithBatchTimeout(10*time.Second),  //nolint:mnd
			sdktrace.WithExportTimeout(10*time.Second), //nolint:mnd
		)),
		sdktrace.WithResource(c.resource),
		sdktrace.WithSampler(c.traceSampler),
	)

	c.tracerProvider = tp
	otel.SetTracerProvider(tp)

	return nil
}

func (c *Client) configureMeterProvider(ctx context.Context) error {
	if c.meterProvider != nil {
		otel.SetMeterProvider(c.meterProvider)
		return nil
	}

	if c.metricTransport == nil {
		return ErrMetricTransportRequired
	}

	exporter, err := c.metricTransport.GetMetricExporter(ctx)
	if err != nil {
		return err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(15*time.Second)),
		), //nolint:mnd
		sdkmetric.WithResource(c.resource),
	)

	c.meterProvider = mp
	otel.SetMeterProvider(mp)

	return nil
}

func (c *Client) configureLoggerProvider(ctx context.Context) error {
	if c.loggerProvider != nil {
		logglobal.SetLoggerProvider(c.loggerProvider)
		return nil
	}

	if c.logTransport == nil {
		return ErrLogTransportRequired
	}

	exporter, err := c.logTransport.GetLogExporter(ctx)
	if err != nil {
		return err
	}

	queueSize := queueSize()

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(
			exporter,
			sdklog.WithMaxQueueSize(queueSize),
			sdklog.WithExportMaxBatchSize(queueSize),
			sdklog.WithExportInterval(10*time.Second), //nolint:mnd
			sdklog.WithExportTimeout(10*time.Second),  //nolint:mnd
		)),
		sdklog.WithResource(c.resource),
	)

	c.loggerProvider = lp
	logglobal.SetLoggerProvider(lp)
	return nil
}

func (c *Client) Shutdown(ctx context.Context) (err error) {
	c.mu.Lock()
	if c.shutdown {
		c.mu.Unlock()
		return nil
	}
	c.shutdown = true
	c.mu.Unlock()

	kratoslog.Infof("OTLP client is shutting down")

	for _, provider := range []any{
		c.tracerProvider,
		c.meterProvider,
		c.loggerProvider,
	} {
		if provider == nil {
			continue
		}
		if p, ok := provider.(interface {
			Shutdown(context.Context) error
		}); ok {
			if e := p.Shutdown(ctx); e != nil {
				err = errors.Join(err, e)
			}
		}
	}

	return err
}

func (c *Client) runConfiguredHooks(ctx context.Context) error {
	for _, hook := range c.hooks {
		if err := hook.Configured(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) signalEnabled(signal Signal) bool {
	return c.signals&signal != 0
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
