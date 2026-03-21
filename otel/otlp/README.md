# OTLP Configuration

This package provides a configuration for the OpenTelemetry Protocol (OTLP) exporter.

## Installation

```shell
go get github.com/go-fries/fries/otel/otlp/v3
```

## Quick Start

Use `NewGRPCTransport(...)` or `NewHTTPTransport(...)` depending on your collector endpoint.

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v3"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	ctx := context.TODO()

	// gRPC transport
	transport := otlp.NewGRPCTransport("localhost:4317",
		otlp.WithGRPCTransportInsecure(true),
	)

	// client
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

## HTTP Example

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v3"
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

`WithHooks(...)` appends custom hooks to the default hooks.

```go
client, err := otlp.NewClient(
	transport,
	otlp.WithHooks(myHook{}),
)
```
