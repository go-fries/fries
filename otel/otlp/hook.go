package otlp

import (
	"context"

	hostmetrics "go.opentelemetry.io/contrib/instrumentation/host"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
)

// DefaultHooks are the hooks that are enabled by default.
var DefaultHooks = []Hook{
	&RuntimeMetricsHook{},
	&HostMetricsHook{},
}

type Hook interface {
	// Configured is called after the client is fully configured.
	Configured(ctx context.Context, client *Client) error
}

// RuntimeMetricsHook is a hook that starts the runtime metrics collection.
type RuntimeMetricsHook struct{}

func (r *RuntimeMetricsHook) Configured(context.Context, *Client) error {
	return runtimemetrics.Start()
}

// HostMetricsHook is a hook that starts the host metrics collection.
type HostMetricsHook struct{}

func (h *HostMetricsHook) Configured(context.Context, *Client) error {
	return hostmetrics.Start()
}
