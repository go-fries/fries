package semconv

import (
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

const serviceHeader = "x-md-service-name"

var _ transport.Transporter = (*mockTransport)(nil)

type headerCarrier http.Header

func (hc headerCarrier) Get(key string) string {
	return http.Header(hc).Get(key)
}

func (hc headerCarrier) Set(key, value string) {
	http.Header(hc).Set(key, value)
}

func (hc headerCarrier) Add(key, value string) {
	http.Header(hc).Add(key, value)
}

func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range http.Header(hc) {
		keys = append(keys, k)
	}
	return keys
}

func (hc headerCarrier) Values(key string) []string {
	return http.Header(hc).Values(key)
}

type mockTransport struct {
	kind      transport.Kind
	endpoint  string
	operation string
	header    headerCarrier
	request   *http.Request
	route     string
}

func (tr *mockTransport) Kind() transport.Kind            { return tr.kind }
func (tr *mockTransport) Endpoint() string                { return tr.endpoint }
func (tr *mockTransport) Operation() string               { return tr.operation }
func (tr *mockTransport) RequestHeader() transport.Header { return tr.header }
func (tr *mockTransport) ReplyHeader() transport.Header   { return tr.header }
func (tr *mockTransport) Request() *http.Request {
	if tr.request == nil {
		rq, _ := http.NewRequest(http.MethodGet, "/endpoint", nil)
		return rq
	}
	return tr.request
}
func (tr *mockTransport) PathTemplate() string { return tr.route }

func TestBuilderClientHTTP(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://api.example.com:443/v1/items/1", strings.NewReader("payload"))
	require.NoError(t, err)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("User-Agent", "go-test")
	msg := &emptypb.Empty{}

	ctx := transport.NewClientContext(t.Context(), &mockTransport{
		kind:      transport.KindHTTP,
		operation: "/example.Service/Get",
		request:   req,
		route:     "/v1/items/{id}",
	})

	got := NewBuilder(serviceHeader).Client(ctx, msg)
	want := []attribute.KeyValue{
		otelsemconv.HTTPRequestMethodGet,
		otelsemconv.URLFull("https://api.example.com:443/v1/items/1"),
		otelsemconv.HTTPRequestBodySize(7),
		otelsemconv.ServerAddress("api.example.com"),
		otelsemconv.ServerPort(443),
		otelsemconv.UserAgentOriginal("go-test"),
		otelsemconv.RPCSystemNameKey.String(transport.KindHTTP.String()),
		otelsemconv.RPCMethod("example.Service/Get"),
		attribute.Key("send_msg.size").Int(proto.Size(msg)),
	}

	assert.Equal(t, want, got)
}

func TestBuilderClientGRPC(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		serverAttrs []attribute.KeyValue
	}{
		{
			name:     "bare host port",
			endpoint: "example.com:9000",
			serverAttrs: []attribute.KeyValue{
				otelsemconv.ServerAddress("example.com"),
				otelsemconv.ServerPort(9000),
			},
		},
		{
			name:     "dns target",
			endpoint: "dns:///example.com:9000",
			serverAttrs: []attribute.KeyValue{
				otelsemconv.ServerAddress("example.com"),
				otelsemconv.ServerPort(9000),
			},
		},
		{
			name:     "discovery target",
			endpoint: "discovery:///user-service",
			serverAttrs: []attribute.KeyValue{
				otelsemconv.ServerAddress("user-service"),
			},
		},
		{
			name:     "target with authority",
			endpoint: "dns://resolver.example.com/example.com:9000",
			serverAttrs: []attribute.KeyValue{
				otelsemconv.ServerAddress("example.com"),
				otelsemconv.ServerPort(9000),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &emptypb.Empty{}
			ip, err := net.ResolveIPAddr("ip", "1.1.1.1")
			require.NoError(t, err)
			ctx := peer.NewContext(t.Context(), &peer.Peer{Addr: ip})
			ctx = transport.NewClientContext(ctx, &mockTransport{
				kind:      transport.KindGRPC,
				endpoint:  tt.endpoint,
				operation: "/example.Service/Get",
			})

			got := NewBuilder(serviceHeader).Client(ctx, msg)
			want := append([]attribute.KeyValue{}, tt.serverAttrs...)
			want = append(
				want,
				otelsemconv.RPCSystemNameGRPC,
				otelsemconv.RPCMethod("example.Service/Get"),
				attribute.Key("send_msg.size").Int(proto.Size(msg)),
			)

			assert.Equal(t, want, got)
		})
	}
}

func TestBuilderServerHTTP(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost/v1/items", strings.NewReader("payload"))
	require.NoError(t, err)
	req.RemoteAddr = "192.168.0.10:54321"
	req.Header.Set("User-Agent", "go-test")
	msg := &emptypb.Empty{}

	ctx := metadata.NewServerContext(t.Context(), metadata.New(map[string][]string{
		serviceHeader: {"caller-service"},
	}))
	ctx = transport.NewServerContext(ctx, &mockTransport{
		kind:      transport.KindHTTP,
		operation: "/example.Service/Create",
		request:   req,
		route:     "/v1/items",
	})

	got := NewBuilder(serviceHeader).Server(ctx, msg)
	want := []attribute.KeyValue{
		otelsemconv.HTTPRequestMethodPost,
		otelsemconv.URLPath("/v1/items"),
		otelsemconv.URLScheme("http"),
		otelsemconv.HTTPRequestBodySize(7),
		otelsemconv.HTTPRoute("/v1/items"),
		otelsemconv.ServerAddress("localhost"),
		otelsemconv.ServerPort(80),
		otelsemconv.ClientAddress("192.168.0.10"),
		otelsemconv.UserAgentOriginal("go-test"),
		otelsemconv.RPCSystemNameKey.String(transport.KindHTTP.String()),
		otelsemconv.RPCMethod("example.Service/Create"),
		otelsemconv.NetworkPeerAddress("192.168.0.10"),
		otelsemconv.NetworkPeerPort(54321),
		attribute.Key("recv_msg.size").Int(proto.Size(msg)),
		otelsemconv.ServicePeerName("caller-service"),
	}

	assert.Equal(t, want, got)
}

func TestBuilderServerGRPC(t *testing.T) {
	msg := &emptypb.Empty{}
	ip, err := net.ResolveIPAddr("ip", "1.1.1.1")
	require.NoError(t, err)
	ctx := peer.NewContext(t.Context(), &peer.Peer{Addr: ip})
	ctx = transport.NewServerContext(ctx, &mockTransport{
		kind:      transport.KindGRPC,
		operation: "/example.Service/Get",
	})

	got := NewBuilder(serviceHeader).Server(ctx, msg)
	want := []attribute.KeyValue{
		otelsemconv.RPCSystemNameGRPC,
		otelsemconv.RPCMethod("example.Service/Get"),
		attribute.Key("recv_msg.size").Int(proto.Size(msg)),
	}

	assert.Equal(t, want, got)
}
