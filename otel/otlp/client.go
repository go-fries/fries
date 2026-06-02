package otlp

import (
	"context"
	"errors"
	"sync"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
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
	otel.SetTextMapPropagator(c.config.newTextMapPropagator())
}

func (c *Client) configureTraceProvider(ctx context.Context) error {
	provider, err := c.config.newTracerProvider(ctx)
	if err != nil {
		return err
	}

	c.config.tracerProvider = provider
	otel.SetTracerProvider(provider)

	return nil
}

func (c *Client) configureMeterProvider(ctx context.Context) error {
	provider, err := c.config.newMeterProvider(ctx)
	if err != nil {
		return err
	}

	c.config.meterProvider = provider
	otel.SetMeterProvider(provider)

	return nil
}

func (c *Client) configureLoggerProvider(ctx context.Context) error {
	provider, err := c.config.newLoggerProvider(ctx)
	if err != nil {
		return err
	}

	c.config.loggerProvider = provider
	logglobal.SetLoggerProvider(provider)
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
