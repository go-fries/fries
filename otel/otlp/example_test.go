package otlp_test

import (
	"context"

	"github.com/go-fries/fries/otel/otlp/v3"
	"go.opentelemetry.io/otel/attribute"
)

func ExampleNewClient() {
	ctx := context.TODO()

	transport := otlp.NewGRPCTransport(
		"localhost:4317",
		otlp.WithGRPCTransportInsecure(true),
	)

	client, err := otlp.NewClient(
		transport,
		otlp.WithServiceName("service-name"),
		otlp.WithDeploymentEnvironmentName("production"),
		otlp.WithAttributes(
			attribute.String("key", "value"),
		),
	)
	if err != nil {
		panic(err)
	}

	if err := client.Configure(ctx); err != nil {
		panic(err)
	}

	defer client.Shutdown(ctx) //nolint:errcheck

	// do something
}
