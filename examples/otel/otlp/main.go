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
	client := otlp.NewClient(
		otlp.WithServiceName("service-name"),
		otlp.WithDeploymentEnvironmentName("production"),
		otlp.WithAttributes(
			attribute.String("key", "value"),
			// ...
		),
		otlp.WithTransport(transport),
	)

	if err := client.Configure(ctx); err != nil {
		panic(err)
	}

	defer client.Shutdown(ctx)

	// do something
}
