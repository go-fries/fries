package semconv

import (
	"net/http"
	"testing"

	"github.com/go-fries/fries/hyperf/jet/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestBuilderClientWithoutJetClient(t *testing.T) {
	got := NewBuilder().Client(t.Context(), "example.Service", "Get")

	assert.Equal(t, []attribute.KeyValue{
		otelsemconv.RPCMethod("example.Service/Get"),
	}, got)
}

func TestBuilderClientWithJSONRPCAndHTTPTransporter(t *testing.T) {
	transporter, err := jet.NewHTTPTransporter(
		jet.WithHTTPTransporterAddr("https://api.example.com:9443/rpc"),
		jet.WithHTTPTransporterClient(http.DefaultClient),
	)
	require.NoError(t, err)

	client, err := jet.NewClient(
		jet.WithService("example.Service"),
		jet.WithTransporter(transporter),
	)
	require.NoError(t, err)
	ctx := jet.ContextWithClient(t.Context(), client)

	got := NewBuilder().Client(ctx, "example.Service", "Get")

	assert.Contains(t, got, otelsemconv.RPCMethod("example.Service/Get"))
	assert.Contains(t, got, otelsemconv.RPCSystemNameJSONRPC)
	assert.Contains(t, got, otelsemconv.JSONRPCProtocolVersion(jet.JSONRPCVersion))
	assert.Contains(t, got, otelsemconv.HTTPRequestMethodPost)
	assert.Contains(t, got, otelsemconv.URLFull("https://api.example.com:9443/rpc"))
	assert.Contains(t, got, otelsemconv.ServerAddress("api.example.com"))
	assert.Contains(t, got, otelsemconv.ServerPort(9443))
}

func TestBuilderClientHTTPTransporterDefaultPorts(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want []attribute.KeyValue
	}{
		{
			name: "http",
			addr: "http://api.example.com/rpc",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("api.example.com"),
				otelsemconv.ServerPort(80),
			},
		},
		{
			name: "https",
			addr: "https://api.example.com/rpc",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("api.example.com"),
				otelsemconv.ServerPort(443),
			},
		},
		{
			name: "host port",
			addr: "api.example.com:9000",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("api.example.com"),
				otelsemconv.ServerPort(9000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpTransporterAttributes(&jet.HTTPTransporter{Addr: tt.addr})

			for _, want := range tt.want {
				assert.Contains(t, got, want)
			}
		})
	}
}
