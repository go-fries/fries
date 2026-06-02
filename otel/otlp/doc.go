// Package otlp configures OpenTelemetry global providers with OTLP exporters.
//
// The package supports trace, metric, and log signals. Use [NewClient] when one
// transport can create exporters for every signal, or use [NewTraceClient],
// [NewMetricClient], and [NewLogClient] for single-signal setup.
package otlp
