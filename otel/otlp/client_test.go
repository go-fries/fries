package otlp

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	logglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewClient(t *testing.T) {
	t.Run("returns error without transport", func(t *testing.T) {
		client, err := NewClient(nil)

		require.Nil(t, client)
		require.ErrorIs(t, err, ErrTransportRequired)
	})
}

func TestClientLifecycle(t *testing.T) {
	t.Run("shutdown managed providers", func(t *testing.T) {
		restoreGlobals := saveGlobalProviders(t)
		defer restoreGlobals()

		traceExporter := &testTraceExporter{}
		metricExporter := &testMetricExporter{}
		logExporter := &testLogExporter{}

		client := newTestClient(t,
			&testTransport{
				traceExporter:  traceExporter,
				metricExporter: metricExporter,
				logExporter:    logExporter,
			},
			WithHooks(noopHook{}),
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
	})

	t.Run("configure twice", func(t *testing.T) {
		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithHooks(noopHook{}),
		)

		require.NoError(t, client.Configure(t.Context()))
		require.ErrorIs(t, client.Configure(t.Context()), ErrClientConfigured)
	})

	t.Run("configure after shutdown", func(t *testing.T) {
		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithHooks(noopHook{}),
		)

		require.NoError(t, client.Shutdown(t.Context()))
		require.ErrorIs(t, client.Configure(t.Context()), ErrClientShutdown)
	})

	t.Run("shutdown is idempotent", func(t *testing.T) {
		restoreGlobals := saveGlobalProviders(t)
		defer restoreGlobals()

		traceExporter := &testTraceExporter{}
		metricExporter := &testMetricExporter{}
		logExporter := &testLogExporter{}

		client := newTestClient(t,
			&testTransport{
				traceExporter:  traceExporter,
				metricExporter: metricExporter,
				logExporter:    logExporter,
			},
			WithHooks(noopHook{}),
		)

		require.NoError(t, client.Configure(t.Context()))
		require.NoError(t, client.Shutdown(t.Context()))
		require.NoError(t, client.Shutdown(t.Context()))

		assert.Equal(t, int32(1), traceExporter.shutdownCount.Load())
		assert.Equal(t, int32(1), metricExporter.shutdownCount.Load())
		assert.Equal(t, int32(1), logExporter.shutdownCount.Load())
	})
}

func TestClientHooks(t *testing.T) {
	t.Run("with hooks appends to defaults", func(t *testing.T) {
		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithHooks(noopHook{}),
		)

		require.Len(t, client.hooks, len(DefaultHooks())+1)
		assert.IsType(t, RuntimeMetricsHook{}, client.hooks[0])
		assert.IsType(t, HostMetricsHook{}, client.hooks[1])
		assert.IsType(t, noopHook{}, client.hooks[2])
	})

	t.Run("default hooks returns copy", func(t *testing.T) {
		hooks := DefaultHooks()
		require.Len(t, hooks, 2)

		hooks[0] = noopHook{}

		otherHooks := DefaultHooks()
		require.Len(t, otherHooks, 2)
		assert.IsType(t, RuntimeMetricsHook{}, otherHooks[0])
		assert.IsType(t, HostMetricsHook{}, otherHooks[1])
	})
}

func TestClientOptions(t *testing.T) {
	t.Run("resource and attributes", func(t *testing.T) {
		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithServiceName("service-name"),
			WithDeploymentEnvironmentName("production"),
			WithAttributes(attribute.String("key", "value")),
		)

		require.NoError(t, client.configureResource(t.Context()))
		require.NotNil(t, client.resource)

		assert.Equal(t, "service-name", client.serviceName)
		assert.Equal(t, "production", client.deploymentEnvironmentName)
		assert.Len(t, client.attributes, 1)
	})

	t.Run("custom providers and propagator", func(t *testing.T) {
		restoreGlobals := saveGlobalProviders(t)
		defer restoreGlobals()

		tracerProvider := sdktrace.NewTracerProvider()
		meterProvider := sdkmetric.NewMeterProvider()
		loggerProvider := sdklog.NewLoggerProvider()
		propagator := propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
		ctx := t.Context()

		t.Cleanup(func() {
			require.NoError(t, tracerProvider.Shutdown(ctx))
			require.NoError(t, meterProvider.Shutdown(ctx))
			require.NoError(t, loggerProvider.Shutdown(ctx))
		})

		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithTracerProvider(tracerProvider),
			WithMeterProvider(meterProvider),
			WithLoggerProvider(loggerProvider),
			WithPropagator(propagator),
		)

		require.NoError(t, client.Configure(ctx))

		assert.Same(t, tracerProvider, client.tracerProvider)
		assert.Same(t, meterProvider, client.meterProvider)
		assert.Same(t, loggerProvider, client.loggerProvider)
		assert.NotNil(t, client.propagator)
		assert.IsType(t, tracerProvider, otel.GetTracerProvider())
		assert.IsType(t, meterProvider, otel.GetMeterProvider())
		assert.IsType(t, loggerProvider, logglobal.GetLoggerProvider())
	})

	t.Run("trace sampler", func(t *testing.T) {
		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithTraceSampler(sdktrace.NeverSample()),
		)

		assert.NotNil(t, client.traceSampler)
	})
}

func TestProvider(t *testing.T) {
	t.Run("bootstrap configures client", func(t *testing.T) {
		restoreGlobals := saveGlobalProviders(t)
		defer restoreGlobals()

		client := newTestClient(t,
			&testTransport{
				traceExporter:  &testTraceExporter{},
				metricExporter: &testMetricExporter{},
				logExporter:    &testLogExporter{},
			},
			WithHooks(noopHook{}),
		)

		provider := NewProvider(client)
		ctx, err := provider.Bootstrap(t.Context())
		require.NoError(t, err)
		assert.Equal(t, t.Context(), ctx)
		assert.True(t, client.configured)
	})

	t.Run("terminate shuts down client", func(t *testing.T) {
		restoreGlobals := saveGlobalProviders(t)
		defer restoreGlobals()

		traceExporter := &testTraceExporter{}
		metricExporter := &testMetricExporter{}
		logExporter := &testLogExporter{}

		client := newTestClient(t,
			&testTransport{
				traceExporter:  traceExporter,
				metricExporter: metricExporter,
				logExporter:    logExporter,
			},
			WithHooks(noopHook{}),
		)
		require.NoError(t, client.Configure(t.Context()))

		provider := NewProvider(client)
		ctx, err := provider.Terminate(t.Context())
		require.NoError(t, err)
		assert.Equal(t, t.Context(), ctx)
		assert.True(t, client.shutdown)
		assert.Equal(t, int32(1), traceExporter.shutdownCount.Load())
		assert.Equal(t, int32(1), metricExporter.shutdownCount.Load())
		assert.Equal(t, int32(1), logExporter.shutdownCount.Load())
	})
}

func newTestClient(t *testing.T, transport Transport, opts ...Option) *Client {
	t.Helper()

	clientOpts := append([]Option{WithResource(sdkresource.Empty())}, opts...)
	client, err := NewClient(transport, clientOpts...)
	require.NoError(t, err)

	return client
}

func saveGlobalProviders(t *testing.T) func() {
	t.Helper()

	oldTracerProvider := otel.GetTracerProvider()
	oldMeterProvider := otel.GetMeterProvider()
	oldLoggerProvider := logglobal.GetLoggerProvider()

	return func() {
		otel.SetTracerProvider(oldTracerProvider)
		otel.SetMeterProvider(oldMeterProvider)
		logglobal.SetLoggerProvider(oldLoggerProvider)
	}
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
