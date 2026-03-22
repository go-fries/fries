package otlp

import (
	"context"

	hostmetrics "go.opentelemetry.io/contrib/instrumentation/host"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
)

var defaultHooks = []Hook{
	RuntimeMetricsHook{},
	HostMetricsHook{},
}

// DefaultHooks returns the hooks that are enabled by default.
func DefaultHooks() []Hook {
	return append([]Hook(nil), defaultHooks...)
}

type Hook interface {
	// Configured is called after the client is fully configured.
	Configured(ctx context.Context, client *Client) error
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
