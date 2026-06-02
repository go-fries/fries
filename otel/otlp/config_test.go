package otlp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestNewConfig(t *testing.T) {
	cfg := newConfig(allSignals)

	assert.True(t, cfg.signalEnabled(TraceSignal))
	assert.True(t, cfg.signalEnabled(MetricSignal))
	assert.True(t, cfg.signalEnabled(LogSignal))
	assert.Equal(t, defaultTraceBatchTimeout, cfg.traceBatchTimeout)
	assert.Equal(t, defaultTraceExportTimeout, cfg.traceExportTimeout)
	assert.Equal(t, defaultMetricInterval, cfg.metricInterval)
	assert.Equal(t, defaultLogExportInterval, cfg.logExportInterval)
	assert.Equal(t, defaultLogExportTimeout, cfg.logExportTimeout)
	assert.Positive(t, cfg.batchQueueSize)
}

func TestConfigResourceOptions(t *testing.T) {
	t.Run("creates resource from attributes", func(t *testing.T) {
		cfg := newConfig(
			allSignals,
			WithServiceName("service-name"),
			WithDeploymentEnvironmentName("production"),
			WithAttributes(attribute.String("key", "value")),
		)

		res, err := cfg.newResource(t.Context())
		require.NoError(t, err)
		require.NotNil(t, res)

		assert.Equal(t, "service-name", cfg.serviceName)
		assert.Equal(t, "production", cfg.deploymentEnvironmentName)
		assert.Equal(t, []attribute.KeyValue{attribute.String("key", "value")}, cfg.attributes)
		assert.Contains(t, res.Attributes(), attribute.String("service.name", "service-name"))
		assert.Contains(t, res.Attributes(), attribute.String("deployment.environment.name", "production"))
		assert.Contains(t, res.Attributes(), attribute.String("key", "value"))
	})

	t.Run("uses configured resource", func(t *testing.T) {
		resource := sdkresource.Empty()
		cfg := newConfig(allSignals, WithResource(resource))

		res, err := cfg.newResource(t.Context())
		require.NoError(t, err)

		assert.Same(t, resource, res)
	})

	t.Run("nil resource falls back to default", func(t *testing.T) {
		cfg := newConfig(allSignals, WithResource(nil))

		res, err := cfg.newResource(t.Context())
		require.NoError(t, err)

		assert.NotNil(t, res)
	})
}

func TestConfigCoreOptions(t *testing.T) {
	resource := sdkresource.Empty()
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	tracerProvider := sdktrace.NewTracerProvider()
	meterProvider := sdkmetric.NewMeterProvider()
	loggerProvider := sdklog.NewLoggerProvider()
	sampler := sdktrace.NeverSample()

	ctx := t.Context()
	t.Cleanup(func() {
		require.NoError(t, tracerProvider.Shutdown(ctx))
		require.NoError(t, meterProvider.Shutdown(ctx))
		require.NoError(t, loggerProvider.Shutdown(ctx))
	})

	cfg := newConfig(
		allSignals,
		WithResource(resource),
		WithPropagator(propagator),
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
		WithLoggerProvider(loggerProvider),
		WithTraceSampler(sampler),
	)

	assert.Same(t, resource, cfg.resource)
	assert.Equal(t, propagator, cfg.newTextMapPropagator())
	assert.Same(t, tracerProvider, cfg.tracerProvider)
	assert.Same(t, meterProvider, cfg.meterProvider)
	assert.Same(t, loggerProvider, cfg.loggerProvider)
	assert.Equal(t, sampler, cfg.traceSampler)
}

func TestConfigBatchOptions(t *testing.T) {
	t.Run("sets valid values", func(t *testing.T) {
		cfg := newConfig(
			allSignals,
			WithBatchQueueSize(256),
			WithTraceBatchTimeout(time.Second),
			WithTraceExportTimeout(2*time.Second),
			WithMetricInterval(3*time.Second),
			WithLogExportInterval(4*time.Second),
			WithLogExportTimeout(5*time.Second),
		)

		assert.Equal(t, 256, cfg.batchQueueSize)
		assert.Equal(t, time.Second, cfg.traceBatchTimeout)
		assert.Equal(t, 2*time.Second, cfg.traceExportTimeout)
		assert.Equal(t, 3*time.Second, cfg.metricInterval)
		assert.Equal(t, 4*time.Second, cfg.logExportInterval)
		assert.Equal(t, 5*time.Second, cfg.logExportTimeout)
	})

	t.Run("ignores invalid values", func(t *testing.T) {
		cfg := newConfig(
			allSignals,
			WithBatchQueueSize(0),
			WithTraceBatchTimeout(0),
			WithTraceExportTimeout(0),
			WithMetricInterval(0),
			WithLogExportInterval(0),
			WithLogExportTimeout(0),
		)

		assert.Equal(t, defaultTraceBatchTimeout, cfg.traceBatchTimeout)
		assert.Equal(t, defaultTraceExportTimeout, cfg.traceExportTimeout)
		assert.Equal(t, defaultMetricInterval, cfg.metricInterval)
		assert.Equal(t, defaultLogExportInterval, cfg.logExportInterval)
		assert.Equal(t, defaultLogExportTimeout, cfg.logExportTimeout)
		assert.Positive(t, cfg.batchQueueSize)
	})
}

func TestConfigSignals(t *testing.T) {
	t.Run("combines signals", func(t *testing.T) {
		cfg := newConfig(allSignals, WithSignals(TraceSignal, LogSignal))

		assert.True(t, cfg.signalEnabled(TraceSignal))
		assert.False(t, cfg.signalEnabled(MetricSignal))
		assert.True(t, cfg.signalEnabled(LogSignal))
	})

	t.Run("keeps empty signal set empty", func(t *testing.T) {
		cfg := newConfig(allSignals, WithSignals())

		assert.False(t, cfg.signalEnabled(TraceSignal))
		assert.False(t, cfg.signalEnabled(MetricSignal))
		assert.False(t, cfg.signalEnabled(LogSignal))
	})
}

func TestConfigHooks(t *testing.T) {
	t.Run("appends custom hooks", func(t *testing.T) {
		cfg := newConfig(allSignals, WithHooks(noopHook{}))

		require.Len(t, cfg.hooks, 1)
		assert.IsType(t, noopHook{}, cfg.hooks[0])
	})

	t.Run("runtime and host metrics are explicit", func(t *testing.T) {
		cfg := newConfig(allSignals, WithRuntimeMetrics(), WithHostMetrics())

		require.Len(t, cfg.hooks, 2)
		assert.IsType(t, RuntimeMetricsHook{}, cfg.hooks[0])
		assert.IsType(t, HostMetricsHook{}, cfg.hooks[1])
	})
}

func TestConfigProviders(t *testing.T) {
	t.Run("uses configured providers", func(t *testing.T) {
		tracerProvider := sdktrace.NewTracerProvider()
		meterProvider := sdkmetric.NewMeterProvider()
		loggerProvider := sdklog.NewLoggerProvider()

		ctx := t.Context()
		t.Cleanup(func() {
			require.NoError(t, tracerProvider.Shutdown(ctx))
			require.NoError(t, meterProvider.Shutdown(ctx))
			require.NoError(t, loggerProvider.Shutdown(ctx))
		})

		cfg := newConfig(
			allSignals,
			WithResource(sdkresource.Empty()),
			WithTracerProvider(tracerProvider),
			WithMeterProvider(meterProvider),
			WithLoggerProvider(loggerProvider),
		)

		traceProvider, err := cfg.newTracerProvider(ctx)
		require.NoError(t, err)
		meterProviderResult, err := cfg.newMeterProvider(ctx)
		require.NoError(t, err)
		loggerProviderResult, err := cfg.newLoggerProvider(ctx)
		require.NoError(t, err)

		assert.Same(t, tracerProvider, traceProvider)
		assert.Same(t, meterProvider, meterProviderResult)
		assert.Same(t, loggerProvider, loggerProviderResult)
	})

	t.Run("requires enabled transports", func(t *testing.T) {
		cfg := newConfig(allSignals, WithResource(sdkresource.Empty()))

		_, err := cfg.newTracerProvider(t.Context())
		require.ErrorIs(t, err, ErrTraceTransportRequired)
		_, err = cfg.newMeterProvider(t.Context())
		require.ErrorIs(t, err, ErrMetricTransportRequired)
		_, err = cfg.newLoggerProvider(t.Context())
		require.ErrorIs(t, err, ErrLogTransportRequired)
	})

	t.Run("creates providers from transports", func(t *testing.T) {
		cfg := newConfig(allSignals, WithResource(sdkresource.Empty()))
		cfg.traceTransport = &testTransport{traceExporter: &testTraceExporter{}}
		cfg.metricTransport = &testTransport{metricExporter: &testMetricExporter{}}
		cfg.logTransport = &testTransport{logExporter: &testLogExporter{}}

		traceProvider, err := cfg.newTracerProvider(t.Context())
		require.NoError(t, err)
		cleanupProvider(t, traceProvider)
		meterProvider, err := cfg.newMeterProvider(t.Context())
		require.NoError(t, err)
		cleanupProvider(t, meterProvider)
		loggerProvider, err := cfg.newLoggerProvider(t.Context())
		require.NoError(t, err)
		cleanupProvider(t, loggerProvider)

		assert.NotNil(t, traceProvider)
		assert.NotNil(t, meterProvider)
		assert.NotNil(t, loggerProvider)
	})
}

func cleanupProvider(t *testing.T, provider any) {
	t.Helper()

	shutdownProvider, ok := provider.(interface {
		Shutdown(context.Context) error
	})
	if !ok {
		return
	}

	t.Cleanup(func() {
		ctx, cancel := newCleanupContext()
		defer cancel()

		require.NoError(t, shutdownProvider.Shutdown(ctx))
	})
}

func newCleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Second)
}

func TestConfigTextMapPropagator(t *testing.T) {
	t.Run("uses default propagator", func(t *testing.T) {
		cfg := newConfig(allSignals)

		assert.NotNil(t, cfg.newTextMapPropagator())
	})

	t.Run("nil propagator falls back to default", func(t *testing.T) {
		cfg := newConfig(allSignals, WithPropagator(nil))

		assert.NotNil(t, cfg.newTextMapPropagator())
	})
}
