package otlp

import (
	"context"
	"errors"
	"runtime"
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
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	// otlp transport
	transport Transport

	// core components
	resource       *sdkresource.Resource
	propagator     propagation.TextMapPropagator
	tracerProvider trace.TracerProvider
	meterProvider  metric.MeterProvider
	loggerProvider log.LoggerProvider

	// resource options
	serviceName           string
	deploymentEnvironment string
	attributes            []attribute.KeyValue

	// trance options
	traceSampler sdktrace.Sampler // default is always on

	// hooks
	hooks []Hook
}

type Option func(*Client)

func WithTransport(transport Transport) Option {
	return func(c *Client) {
		c.transport = transport
	}
}

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

func WithServiceName(serviceName string) Option {
	return func(c *Client) {
		c.serviceName = serviceName
	}
}

func WithDeploymentEnvironment(deploymentEnvironment string) Option {
	return func(c *Client) {
		c.deploymentEnvironment = deploymentEnvironment
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

func WithHook(hooks ...Hook) Option {
	return func(c *Client) {
		if len(hooks) > 0 {
			c.hooks = hooks
		}
	}
}

func NewClient(opts ...Option) *Client {
	c := &Client{
		hooks: DefaultHooks,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Configure(ctx context.Context) error {
	// resource
	if err := c.configureResource(ctx); err != nil {
		return err
	}

	// propagator
	c.configureTextMapPropagator()

	// trace
	if err := c.configureTraceProvider(ctx); err != nil {
		return err
	}

	// metrics
	if err := c.configureMeterProvider(ctx); err != nil {
		return err
	}

	// logger
	if err := c.configureLoggerProvider(ctx); err != nil {
		return err
	}

	// run configured hooks
	if err := c.runConfiguredHooks(ctx); err != nil {
		return err
	}

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

	if c.deploymentEnvironment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironment(c.deploymentEnvironment))
	}

	res, err := sdkresource.New(ctx,
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

	exporter, err := c.transport.GetTraceSpanExporter(ctx)
	if err != nil {
		return err
	}

	queueSize := queueSize()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter,
			sdktrace.WithMaxQueueSize(queueSize),
			sdktrace.WithMaxExportBatchSize(queueSize),
			sdktrace.WithBatchTimeout(10*time.Second),  // nolint:mnd
			sdktrace.WithExportTimeout(10*time.Second), // nolint:mnd
		)),
		sdktrace.WithResource(c.resource),
		sdktrace.WithSampler(c.traceSampler),
	)

	otel.SetTracerProvider(tp)

	return nil
}

func (c *Client) configureMeterProvider(ctx context.Context) error {
	if c.meterProvider != nil {
		otel.SetMeterProvider(c.meterProvider)
		return nil
	}

	exporter, err := c.transport.GetMetricExporter(ctx)
	if err != nil {
		return err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(15*time.Second))), // nolint:mnd
		sdkmetric.WithResource(c.resource),
	)

	otel.SetMeterProvider(mp)

	return nil
}

func (c *Client) configureLoggerProvider(ctx context.Context) error {
	if c.loggerProvider != nil {
		logglobal.SetLoggerProvider(c.loggerProvider)
		return nil
	}

	exporter, err := c.transport.GetLogExporter(ctx)
	if err != nil {
		return err
	}

	queueSize := queueSize()

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter,
			sdklog.WithMaxQueueSize(queueSize),
			sdklog.WithExportMaxBatchSize(queueSize),
			sdklog.WithExportInterval(10*time.Second), // nolint:mnd
			sdklog.WithExportTimeout(10*time.Second),  // nolint:mnd
		)),
		sdklog.WithResource(c.resource),
	)

	logglobal.SetLoggerProvider(lp)
	return nil
}

func (c *Client) Shutdown(ctx context.Context) (err error) {
	kratoslog.Infof("OTLP client shutdowning")

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

func (c *Client) RegisterResource(resource *sdkresource.Resource) {
	c.resource = resource
}

func (c *Client) runConfiguredHooks(ctx context.Context) error {
	for _, hook := range c.hooks {
		if err := hook.Configured(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func queueSize() int {
	const min = 1000  // nolint:mnd
	const max = 16000 // nolint:mnd

	n := (runtime.GOMAXPROCS(0) / 2) * 1000 // nolint:mnd
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
