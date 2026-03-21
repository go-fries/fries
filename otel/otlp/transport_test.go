package otlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		insecure  bool
		newTarget func() transportTarget
	}{
		{
			name:     "grpc",
			endpoint: "localhost:4317",
			insecure: true,
			newTarget: func() transportTarget {
				transport := NewGRPCTransport("localhost:4317", WithGRPCTransportInsecure(true))
				return transportTarget{
					endpoint: transport.endpoint,
					insecure: transport.insecure,
					trace: func(ctx context.Context) (any, error) {
						return transport.GetTraceSpanExporter(ctx)
					},
					metric: func(ctx context.Context) (any, error) {
						return transport.GetMetricExporter(ctx)
					},
					log: func(ctx context.Context) (any, error) {
						return transport.GetLogExporter(ctx)
					},
				}
			},
		},
		{
			name:     "http",
			endpoint: "localhost:4318",
			insecure: true,
			newTarget: func() transportTarget {
				transport := NewHTTPTransport("localhost:4318", WithHTTPTransportInsecure(true))
				return transportTarget{
					endpoint: transport.endpoint,
					insecure: transport.insecure,
					trace: func(ctx context.Context) (any, error) {
						return transport.GetTraceSpanExporter(ctx)
					},
					metric: func(ctx context.Context) (any, error) {
						return transport.GetMetricExporter(ctx)
					},
					log: func(ctx context.Context) (any, error) {
						return transport.GetLogExporter(ctx)
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.newTarget()

			assert.Equal(t, tt.endpoint, target.endpoint)
			assert.Equal(t, tt.insecure, target.insecure)

			t.Run("trace exporter", func(t *testing.T) {
				exporter, err := target.trace(t.Context())
				require.NoError(t, err)
				assert.NotNil(t, exporter)
			})

			t.Run("metric exporter", func(t *testing.T) {
				exporter, err := target.metric(t.Context())
				require.NoError(t, err)
				assert.NotNil(t, exporter)
			})

			t.Run("log exporter", func(t *testing.T) {
				exporter, err := target.log(t.Context())
				require.NoError(t, err)
				assert.NotNil(t, exporter)
			})
		})
	}
}

type transportTarget struct {
	endpoint string
	insecure bool
	trace    func(ctx context.Context) (any, error)
	metric   func(ctx context.Context) (any, error)
	log      func(ctx context.Context) (any, error)
}
