package otlp

import (
	"context"
	"errors"
	"sync"

	kratoslog "github.com/go-kratos/kratos/v2/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
)

var (
	// ErrTransportRequired is returned when an all-signal [Transport] is nil.
	ErrTransportRequired = errors.New("otlp transport is required")
	// ErrTraceTransportRequired is returned when a [TraceTransport] is required but nil.
	ErrTraceTransportRequired = errors.New("otlp trace transport is required")
	// ErrMetricTransportRequired is returned when a [MetricTransport] is required but nil.
	ErrMetricTransportRequired = errors.New("otlp metric transport is required")
	// ErrLogTransportRequired is returned when a [LogTransport] is required but nil.
	ErrLogTransportRequired = errors.New("otlp log transport is required")
	// ErrClientConfigured is returned when [Client.Configure] is called more than once.
	ErrClientConfigured = errors.New("otlp client has already been configured")
	// ErrClientShutdown is returned when [Client.Configure] is called after [Client.Shutdown].
	ErrClientShutdown = errors.New("otlp client has been shut down")
)

// Client configures OpenTelemetry global providers backed by OTLP exporters.
//
// A client may be configured once and shut down once. Configure registers the
// selected global providers and Shutdown closes any configured SDK providers
// that support shutdown.
type Client struct {
	mu sync.Mutex

	config config

	configured bool
	shutdown   bool
}

// NewClient creates a [Client] that uses transport for trace, metric, and log signals.
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

// NewTraceClient creates a [Client] that configures only the trace signal.
func NewTraceClient(transport TraceTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrTraceTransportRequired
	}

	cfg := newConfig(TraceSignal, opts...)
	cfg.traceTransport = transport

	return &Client{config: *cfg}, nil
}

// NewMetricClient creates a [Client] that configures only the metric signal.
func NewMetricClient(transport MetricTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrMetricTransportRequired
	}

	cfg := newConfig(MetricSignal, opts...)
	cfg.metricTransport = transport

	return &Client{config: *cfg}, nil
}

// NewLogClient creates a [Client] that configures only the log signal.
func NewLogClient(transport LogTransport, opts ...Option) (*Client, error) {
	if transport == nil {
		return nil, ErrLogTransportRequired
	}

	cfg := newConfig(LogSignal, opts...)
	cfg.logTransport = transport

	return &Client{config: *cfg}, nil
}

// Configure initializes resources, providers, propagators, and configured hooks.
//
// Configure sets OpenTelemetry global providers for the enabled signals. It is
// not idempotent: calling it more than once returns [ErrClientConfigured].
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
	global.SetLoggerProvider(provider)
	return nil
}

// Shutdown shuts down all configured SDK providers that support shutdown.
//
// Shutdown is idempotent. Errors from multiple providers are joined and
// returned as a single error.
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
