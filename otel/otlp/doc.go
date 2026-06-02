// Package otlp configures OpenTelemetry global providers with OTLP exporters.
//
// The package supports trace, metric, and log signals. Use [NewClient] when one
// transport can create exporters for every signal, or use [NewTraceClient],
// [NewMetricClient], and [NewLogClient] for single-signal setup.
//
// Configure all signals with a shared gRPC transport:
//
//	ctx := context.Background()
//	transport := otlp.NewGRPCTransport(
//		"localhost:4317",
//		otlp.WithGRPCTransportInsecure(true),
//	)
//
//	client, err := otlp.NewClient(
//		transport,
//		otlp.WithServiceName("checkout"),
//		otlp.WithDeploymentEnvironmentName("production"),
//	)
//	if err != nil {
//		panic(err)
//	}
//	if err := client.Configure(ctx); err != nil {
//		panic(err)
//	}
//	defer client.Shutdown(ctx) //nolint:errcheck
//
// Configure only tracing when the application does not need metrics or logs:
//
//	client, err := otlp.NewTraceClient(
//		transport,
//		otlp.WithServiceName("checkout"),
//	)
//	if err != nil {
//		panic(err)
//	}
package otlp
