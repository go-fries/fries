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
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Transport interface {
	GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error)
	GetMetricExporter(ctx context.Context) (metric.Exporter, error)
	GetLogExporter(ctx context.Context) (log.Exporter, error)
}

type GRPCTransport struct {
	endpoint string
	insecure bool
}

var _ Transport = (*GRPCTransport)(nil)

type GRPCTransportOption func(*GRPCTransport)

func WithGRPCTransportInsecure(insecure bool) GRPCTransportOption {
	return func(t *GRPCTransport) {
		t.insecure = insecure
	}
}

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

func (t *GRPCTransport) GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(t.endpoint),
		otlptracegrpc.WithCompressor("gzip"),
	}

	if t.insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	return otlptracegrpc.New(ctx, opts...)
}

func (t *GRPCTransport) GetMetricExporter(ctx context.Context) (metric.Exporter, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(t.endpoint),
		otlpmetricgrpc.WithCompressor("gzip"),
		otlpmetricgrpc.WithTemporalitySelector(func(kind metric.InstrumentKind) metricdata.Temporality {
			switch kind {
			case metric.InstrumentKindCounter,
				metric.InstrumentKindObservableCounter,
				metric.InstrumentKindHistogram:
				return metricdata.DeltaTemporality
			default:
				return metricdata.CumulativeTemporality
			}
		}),
	}

	if t.insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	return otlpmetricgrpc.New(ctx, opts...)
}

func (t *GRPCTransport) GetLogExporter(ctx context.Context) (log.Exporter, error) {
	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(t.endpoint),
		otlploggrpc.WithCompressor("gzip"),
	}

	if t.insecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	return otlploggrpc.New(ctx, opts...)
}

type HTTPTransport struct {
	endpoint string
	insecure bool
}

var _ Transport = (*HTTPTransport)(nil)

type HTTPTransportOption func(*HTTPTransport)

func WithHTTPTransportInsecure(insecure bool) HTTPTransportOption {
	return func(t *HTTPTransport) {
		t.insecure = insecure
	}
}

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

func (t *HTTPTransport) GetTraceSpanExporter(ctx context.Context) (trace.SpanExporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(t.endpoint),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if t.insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	return otlptracehttp.New(ctx, opts...)
}

func (t *HTTPTransport) GetMetricExporter(ctx context.Context) (metric.Exporter, error) {
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(t.endpoint),
		otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
		otlpmetrichttp.WithTemporalitySelector(func(kind metric.InstrumentKind) metricdata.Temporality {
			switch kind {
			case metric.InstrumentKindCounter,
				metric.InstrumentKindObservableCounter,
				metric.InstrumentKindHistogram:
				return metricdata.DeltaTemporality
			default:
				return metricdata.CumulativeTemporality
			}
		}),
	}

	if t.insecure {
		opts = append(opts, otlpmetrichttp.WithInsecure())
	}

	return otlpmetrichttp.New(ctx, opts...)
}

func (t *HTTPTransport) GetLogExporter(ctx context.Context) (log.Exporter, error) {
	opts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(t.endpoint),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
	}

	if t.insecure {
		opts = append(opts, otlploghttp.WithInsecure())
	}

	return otlploghttp.New(ctx, opts...)
}
