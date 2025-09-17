package otlp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGRPCTransport(t *testing.T) {
	endpoint := "localhost:4317"
	transport := NewGRPCTransport(endpoint, WithGRPCTransportInsecure(true))

	assert.Equal(t, endpoint, transport.endpoint)
	assert.True(t, transport.insecure)

	ctx := t.Context()

	traceExporter, err := transport.GetTraceSpanExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, traceExporter)

	metricExporter, err := transport.GetMetricExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metricExporter)

	logExporter, err := transport.GetLogExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, logExporter)
}

func TestHTTPTransport(t *testing.T) {
	endpoint := "localhost:4318"
	transport := NewHTTPTransport(endpoint, WithHTTPTransportInsecure(true))

	assert.Equal(t, endpoint, transport.endpoint)
	assert.True(t, transport.insecure)

	ctx := t.Context()

	traceExporter, err := transport.GetTraceSpanExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, traceExporter)

	metricExporter, err := transport.GetMetricExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, metricExporter)

	logExporter, err := transport.GetLogExporter(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, logExporter)
}
