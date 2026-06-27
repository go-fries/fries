# OTLP Configuration

This package provides a configuration for the OpenTelemetry Protocol (OTLP) exporter.

## Installation

```shell
go get github.com/go-fries/fries/otel/otlp/v4
```

## Defaults

- `NewClient(...)` configures trace, metric, and log providers.
- `NewTraceClient(...)`, `NewMetricClient(...)`, and `NewLogClient(...)` configure a single signal.
- Hooks are disabled by default.
- Runtime and host metrics are opt-in.

## All Signals

Use `NewGRPCTransport(...)` or `NewHTTPTransport(...)` depending on your collector endpoint.

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v4"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	ctx := context.TODO()

	// gRPC transport
	transport := otlp.NewGRPCTransport("localhost:4317",
		otlp.WithGRPCTransportInsecure(true),
	)

	client, err := otlp.NewClient(
		transport,
		otlp.WithServiceName("service-name"),
		otlp.WithDeploymentEnvironmentName("production"),
		otlp.WithAttributes(
			attribute.String("key", "value"),
			// ...
		),
	)
	if err != nil {
		panic(err)
	}

	if err := client.Configure(ctx); err != nil {
		panic(err)
	}

	defer client.Shutdown(ctx)

	// do something
}
```

## Tracing Only

Use `NewTraceClient(...)` when only tracing should be configured.

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v4"
)

func main() {
	ctx := context.TODO()

	transport := otlp.NewGRPCTransport("localhost:4317",
		otlp.WithGRPCTransportInsecure(true),
	)

	client, err := otlp.NewTraceClient(
		transport,
		otlp.WithServiceName("service-name"),
	)
	if err != nil {
		panic(err)
	}

	if err := client.Configure(ctx); err != nil {
		panic(err)
	}

	defer client.Shutdown(ctx)
}
```

## Single Signal

Metric-only and log-only clients use the matching transport capability.

```go
metricClient, err := otlp.NewMetricClient(
	transport,
	otlp.WithServiceName("service-name"),
)
```

```go
logClient, err := otlp.NewLogClient(
	transport,
	otlp.WithServiceName("service-name"),
)
```

## HTTP Transport

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v4"
)

func main() {
	ctx := context.TODO()

	transport := otlp.NewHTTPTransport("localhost:4318",
		otlp.WithHTTPTransportInsecure(true),
		otlp.WithHTTPTransportHeaders(map[string]string{
			"authorization": "Bearer <token>",
		}),
	)

	client, err := otlp.NewClient(transport)
	if err != nil {
		panic(err)
	}

	if err := client.Configure(ctx); err != nil {
		panic(err)
	}

	defer client.Shutdown(ctx)
}
```

## Custom Hooks

`WithHooks(...)` appends custom hooks. No hooks are enabled by default.

```go
client, err := otlp.NewClient(
	transport,
	otlp.WithHooks(myHook{}),
)
```

## Runtime and Host Metrics

Runtime and host metrics are opt-in.

```go
client, err := otlp.NewClient(
	transport,
	otlp.WithRuntimeMetrics(),
	otlp.WithHostMetrics(),
)
```

## Batch Options

Batch and reader timing can be tuned with options.

```go
client, err := otlp.NewClient(
	transport,
	otlp.WithBatchQueueSize(2048),
	otlp.WithTraceBatchTimeout(5*time.Second),
	otlp.WithTraceExportTimeout(10*time.Second),
	otlp.WithMetricInterval(15*time.Second),
	otlp.WithLogExportInterval(10*time.Second),
	otlp.WithLogExportTimeout(10*time.Second),
)
```
