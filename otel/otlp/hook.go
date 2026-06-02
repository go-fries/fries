package otlp

import (
	"context"

	hostmetrics "go.opentelemetry.io/contrib/instrumentation/host"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
)

type Hook interface {
	// Configured is called after the client is fully configured.
	Configured(ctx context.Context, client *Client) error
}

// WithRuntimeMetrics starts runtime metrics collection after the client is configured.
func WithRuntimeMetrics() Option {
	return WithHooks(RuntimeMetricsHook{})
}

// WithHostMetrics starts host metrics collection after the client is configured.
func WithHostMetrics() Option {
	return WithHooks(HostMetricsHook{})
}

// RuntimeMetricsHook is a hook that starts the runtime metrics collection.
type RuntimeMetricsHook struct{}

func (RuntimeMetricsHook) Configured(context.Context, *Client) error {
	return runtimemetrics.Start()
}

// HostMetricsHook is a hook that starts the host metrics collection.
type HostMetricsHook struct{}

func (HostMetricsHook) Configured(context.Context, *Client) error {
	return hostmetrics.Start()
}
