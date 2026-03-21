package otlp

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	logglobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestClientShutdownShutsDownManagedProviders(t *testing.T) {
	oldTracerProvider := otel.GetTracerProvider()
	oldMeterProvider := otel.GetMeterProvider()
	oldLoggerProvider := logglobal.GetLoggerProvider()

	t.Cleanup(func() {
		otel.SetTracerProvider(oldTracerProvider)
		otel.SetMeterProvider(oldMeterProvider)
		logglobal.SetLoggerProvider(oldLoggerProvider)
	})

	traceExporter := &testTraceExporter{}
	metricExporter := &testMetricExporter{}
	logExporter := &testLogExporter{}

	client := NewClient(
		WithTransport(&testTransport{
			traceExporter:  traceExporter,
			metricExporter: metricExporter,
			logExporter:    logExporter,
		}),
		WithResource(sdkresource.Empty()),
		WithHook(noopHook{}),
	)

	ctx := t.Context()

	require.NoError(t, client.Configure(ctx))
	require.NotNil(t, client.tracerProvider)
	require.NotNil(t, client.meterProvider)
	require.NotNil(t, client.loggerProvider)

	require.NoError(t, client.Shutdown(ctx))

	assert.Equal(t, int32(1), traceExporter.shutdownCount.Load())
	assert.Equal(t, int32(1), metricExporter.shutdownCount.Load())
	assert.Equal(t, int32(1), logExporter.shutdownCount.Load())
}

type noopHook struct{}

func (noopHook) Configured(context.Context, *Client) error {
	return nil
}

type testTransport struct {
	traceExporter  sdktrace.SpanExporter
	metricExporter sdkmetric.Exporter
	logExporter    sdklog.Exporter
}

func (t *testTransport) GetTraceSpanExporter(context.Context) (sdktrace.SpanExporter, error) {
	return t.traceExporter, nil
}

func (t *testTransport) GetMetricExporter(context.Context) (sdkmetric.Exporter, error) {
	return t.metricExporter, nil
}

func (t *testTransport) GetLogExporter(context.Context) (sdklog.Exporter, error) {
	return t.logExporter, nil
}

type testTraceExporter struct {
	shutdownCount atomic.Int32
}

func (e *testTraceExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *testTraceExporter) Shutdown(context.Context) error {
	e.shutdownCount.Add(1)
	return nil
}

type testMetricExporter struct {
	shutdownCount atomic.Int32
}

func (e *testMetricExporter) Temporality(kind sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *testMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *testMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	return nil
}

func (e *testMetricExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *testMetricExporter) Shutdown(context.Context) error {
	e.shutdownCount.Add(1)
	return nil
}

type testLogExporter struct {
	shutdownCount atomic.Int32
}

func (e *testLogExporter) Export(context.Context, []sdklog.Record) error {
	return nil
}

func (e *testLogExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *testLogExporter) Shutdown(context.Context) error {
	e.shutdownCount.Add(1)
	return nil
}
