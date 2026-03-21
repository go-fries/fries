# OTLP Configuration

This package provides a configuration for the OpenTelemetry Protocol (OTLP) exporter.

## Installation

```shell
go get github.com/go-fries/fries/otel/otlp/v3
```

## Usage Example

```go
package main

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v3"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	ctx := context.TODO()

	// transport
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
