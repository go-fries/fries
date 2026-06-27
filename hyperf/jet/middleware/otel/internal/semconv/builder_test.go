package semconv

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-fries/fries/hyperf/jet/v4"
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
		jet.WithService(`Example\User\MoneyService`),
		jet.WithTransporter(transporter),
	)
	require.NoError(t, err)
	ctx := jet.ContextWithClient(t.Context(), client)

	got := NewBuilder().Client(ctx, `Example\User\MoneyService`, "GetBalance")

	assert.Contains(t, got, otelsemconv.RPCMethod("/money/GetBalance"))
	assert.Contains(t, got, otelsemconv.RPCSystemNameJSONRPC)
	assert.Contains(t, got, otelsemconv.JSONRPCProtocolVersion(jet.JSONRPCVersion))
	assert.NotContains(t, got, otelsemconv.HTTPRequestMethodPost)
	assert.NotContains(t, got, otelsemconv.URLFull("https://api.example.com:9443/rpc"))
	assert.Contains(t, got, otelsemconv.ServerAddress("api.example.com"))
	assert.Contains(t, got, otelsemconv.ServerPort(9443))
}

func TestBuilderOperation(t *testing.T) {
	builder := NewBuilder()

	assert.Equal(t, `Example\User\MoneyService/GetBalance`, builder.Operation(t.Context(), `Example\User\MoneyService`, "GetBalance"))

	client, err := jet.NewClient(
		jet.WithService(`Example\User\MoneyService`),
		jet.WithTransporter(&recordingTransporter{}),
	)
	require.NoError(t, err)
	ctx := jet.ContextWithClient(t.Context(), client)

	assert.Equal(t, "/money/GetBalance", builder.Operation(ctx, `Example\User\MoneyService`, "GetBalance"))
}

type recordingTransporter struct{}

func (*recordingTransporter) Send(context.Context, []byte) ([]byte, error) {
	return nil, nil
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
			name: "host port without scheme",
			addr: "api.example.com:9000",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("api.example.com"),
				otelsemconv.ServerPort(9000),
			},
		},
		{
			name: "ipv6 url",
			addr: "https://[2001:db8::1]:9443/rpc",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("2001:db8::1"),
				otelsemconv.ServerPort(9443),
			},
		},
		{
			name: "ipv6 without explicit port",
			addr: "https://[2001:db8::1]/rpc",
			want: []attribute.KeyValue{
				otelsemconv.ServerAddress("2001:db8::1"),
				otelsemconv.ServerPort(443),
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

func TestBuilderClientHTTPTransporterInvalidPort(t *testing.T) {
	got := httpTransporterAttributes(&jet.HTTPTransporter{Addr: "https://api.example.com:bad/rpc"})

	assert.Empty(t, got)
}
