package otlp

import (
	"context"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// TraceTransport creates an OTLP trace span exporter.
type TraceTransport interface {
	// GetTraceSpanExporter creates the span exporter used by the trace provider.
	GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error)
}

// MetricTransport creates an OTLP metric exporter.
type MetricTransport interface {
	// GetMetricExporter creates the metric exporter used by the meter provider.
	GetMetricExporter(ctx context.Context) (metric.Exporter, error)
}

// LogTransport creates an OTLP log exporter.
type LogTransport interface {
	// GetLogExporter creates the log exporter used by the logger provider.
	GetLogExporter(ctx context.Context) (log.Exporter, error)
}

// Transport creates OTLP exporters for all supported OpenTelemetry signals.
type Transport interface {
	TraceTransport
	MetricTransport
	LogTransport
}

// GRPCTransport creates OTLP exporters over gRPC.
type GRPCTransport struct {
	endpoint string
	insecure bool
	headers  map[string]string
}

var (
	_ Transport       = (*GRPCTransport)(nil)
	_ TraceTransport  = (*GRPCTransport)(nil)
	_ MetricTransport = (*GRPCTransport)(nil)
	_ LogTransport    = (*GRPCTransport)(nil)
)

// GRPCTransportOption configures a [GRPCTransport].
type GRPCTransportOption func(*GRPCTransport)

// WithGRPCTransportInsecure marks gRPC exporter connections as insecure.
func WithGRPCTransportInsecure(insecure bool) GRPCTransportOption {
	return func(t *GRPCTransport) {
		t.insecure = insecure
	}
}

// WithGRPCTransportHeaders sets headers sent by gRPC exporters.
func WithGRPCTransportHeaders(headers map[string]string) GRPCTransportOption {
	return func(t *GRPCTransport) {
		t.headers = headers
	}
}

// NewGRPCTransport creates a [GRPCTransport] with endpoint and options.
func NewGRPCTransport(endpoint string, opts ...GRPCTransportOption) *GRPCTransport {
	transport := &GRPCTransport{
		endpoint: endpoint,
		insecure: false,
	}

	for _, opt := range opts {
		opt(transport)
	}

	return transport
}

// GetTraceSpanExporter creates an OTLP/gRPC trace span exporter.
func (t *GRPCTransport) GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(t.endpoint),
		otlptracegrpc.WithCompressor("gzip"),
	}

	if t.insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlptracegrpc.WithHeaders(t.headers))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// GetMetricExporter creates an OTLP/gRPC metric exporter.
func (t *GRPCTransport) GetMetricExporter(ctx context.Context) (metric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(t.endpoint),
		otlpmetricgrpc.WithCompressor("gzip"),
	}

	if t.insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlpmetricgrpc.WithHeaders(t.headers))
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

// GetLogExporter creates an OTLP/gRPC log exporter.
func (t *GRPCTransport) GetLogExporter(ctx context.Context) (log.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(t.endpoint),
		otlploggrpc.WithCompressor("gzip"),
	}

	if t.insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlploggrpc.WithHeaders(t.headers))
	}

	return otlploggrpc.New(ctx, opts...)
}

// HTTPTransport creates OTLP exporters over HTTP.
type HTTPTransport struct {
	endpoint string
	insecure bool
	headers  map[string]string
}

var (
	_ Transport       = (*HTTPTransport)(nil)
	_ TraceTransport  = (*HTTPTransport)(nil)
	_ MetricTransport = (*HTTPTransport)(nil)
	_ LogTransport    = (*HTTPTransport)(nil)
)

// HTTPTransportOption configures an [HTTPTransport].
type HTTPTransportOption func(*HTTPTransport)

// WithHTTPTransportInsecure marks HTTP exporter connections as insecure.
func WithHTTPTransportInsecure(insecure bool) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.insecure = insecure
	}
}

// WithHTTPTransportHeaders sets headers sent by HTTP exporters.
func WithHTTPTransportHeaders(headers map[string]string) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.headers = headers
	}
}

// NewHTTPTransport creates an [HTTPTransport] with endpoint and options.
func NewHTTPTransport(endpoint string, opts ...HTTPTransportOption) *HTTPTransport {
	transport := &HTTPTransport{
		endpoint: endpoint,
		insecure: false,
	}

	for _, opt := range opts {
		opt(transport)
	}

	return transport
}

// GetTraceSpanExporter creates an OTLP/HTTP trace span exporter.
func (t *HTTPTransport) GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(t.endpoint),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if t.insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlptracehttp.WithHeaders(t.headers))
	}

	return otlptracehttp.New(ctx, opts...)
}

// GetMetricExporter creates an OTLP/HTTP metric exporter.
func (t *HTTPTransport) GetMetricExporter(ctx context.Context) (metric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(t.endpoint),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
	}

	if t.insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlpmetrichttp.WithHeaders(t.headers))
	}

	return otlpmetrichttp.New(ctx, opts...)
}

// GetLogExporter creates an OTLP/HTTP log exporter.
func (t *HTTPTransport) GetLogExporter(ctx context.Context) (log.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(t.endpoint),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
	}

	if t.insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	if t.headers != nil {
		opts = append(opts, otlploghttp.WithHeaders(t.headers))
	}

	return otlploghttp.New(ctx, opts...)
}
