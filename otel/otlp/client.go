package otlp

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	ErrTransportRequired       = errors.New("otlp transport is required")
	ErrTraceTransportRequired  = errors.New("otlp trace transport is required")
	ErrMetricTransportRequired = errors.New("otlp metric transport is required")
	ErrLogTransportRequired    = errors.New("otlp log transport is required")
	ErrClientConfigured        = errors.New("otlp client has already been configured")
	ErrClientShutdown          = errors.New("otlp client has been shut down")
)

type Client struct {
	mu sync.Mutex

	config config

	configured bool
	shutdown   bool
}

// NewClient creates a Client configured with a transport for all signals.
func NewClient(transport Transport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrTransportRequired
	}

	cfg := newConfig(allSignals, opts...)
	cfg.traceTransport = transport
	cfg.metricTransport = transport
	cfg.logTransport = transport

	return &Client{config: *cfg}, nil
}

// NewTraceClient creates a Client configured with a trace transport.
func NewTraceClient(transport TraceTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrTraceTransportRequired
	}

	cfg := newConfig(TraceSignal, opts...)
	cfg.traceTransport = transport

	return &Client{config: *cfg}, nil
}

// NewMetricClient creates a Client configured with a metric transport.
func NewMetricClient(transport MetricTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrMetricTransportRequired
	}

	cfg := newConfig(MetricSignal, opts...)
	cfg.metricTransport = transport

	return &Client{config: *cfg}, nil
}

// NewLogClient creates a Client configured with a log transport.
func NewLogClient(transport LogTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrLogTransportRequired
	}

	cfg := newConfig(LogSignal, opts...)
	cfg.logTransport = transport

	return &Client{config: *cfg}, nil
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

	if c.config.signalEnabled(TraceSignal) {
		if err := c.configureTraceProvider(ctx); err != nil {
			return err
		}
	}

	if c.config.signalEnabled(MetricSignal) {
		if err := c.configureMeterProvider(ctx); err != nil {
			return err
		}
	}

	if c.config.signalEnabled(LogSignal) {
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
	if c.config.resource != nil {
		return nil
	}

	res, err := c.config.newResource(ctx)
	if err != nil {
		return err
	}

	c.config.resource = res

	return nil
}

func (c *Client) configureTextMapPropagator() {
	if c.config.propagator != nil {
		otel.SetTextMapPropagator(c.config.propagator)
		return
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

func (c *Client) configureTraceProvider(ctx context.Context) error {
	if c.config.tracerProvider != nil {
		otel.SetTracerProvider(c.config.tracerProvider)
		return nil
	}

	if c.config.traceTransport == nil {
		return ErrTraceTransportRequired
	}

	exporter, err := c.config.traceTransport.GetTraceSpanExporter(ctx)
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
		sdktrace.WithResource(c.config.resource),
		sdktrace.WithSampler(c.config.traceSampler),
	)

	c.config.tracerProvider = tp
	otel.SetTracerProvider(tp)

	return nil
}

func (c *Client) configureMeterProvider(ctx context.Context) error {
	if c.config.meterProvider != nil {
		otel.SetMeterProvider(c.config.meterProvider)
		return nil
	}

	if c.config.metricTransport == nil {
		return ErrMetricTransportRequired
	}

	exporter, err := c.config.metricTransport.GetMetricExporter(ctx)
	if err != nil {
		return err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(15*time.Second)),
		), //nolint:mnd
		sdkmetric.WithResource(c.config.resource),
	)

	c.config.meterProvider = mp
	otel.SetMeterProvider(mp)

	return nil
}

func (c *Client) configureLoggerProvider(ctx context.Context) error {
	if c.config.loggerProvider != nil {
		logglobal.SetLoggerProvider(c.config.loggerProvider)
		return nil
	}

	if c.config.logTransport == nil {
		return ErrLogTransportRequired
	}

	exporter, err := c.config.logTransport.GetLogExporter(ctx)
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
		sdklog.WithResource(c.config.resource),
	)

	c.config.loggerProvider = lp
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
		c.config.tracerProvider,
		c.config.meterProvider,
		c.config.loggerProvider,
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
	for _, hook := range c.config.hooks {
		if err := hook.Configured(ctx, c); err != nil {
			return err
		}
	}
	return nil
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
