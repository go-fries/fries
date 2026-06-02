package semconv

import (
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestHTTPRequestMethod(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   attribute.KeyValue
	}{
		{
			name:   "known standard method",
			method: "GET",
			want:   otelsemconv.HTTPRequestMethodGet,
		},
		{
			name:   "known semconv method",
			method: "QUERY",
			want:   otelsemconv.HTTPRequestMethodQuery,
		},
		{
			name:   "known semconv other method",
			method: "_OTHER",
			want:   otelsemconv.HTTPRequestMethodOther,
		},
		{
			name:   "unknown method",
			method: "CUSTOM",
			want:   otelsemconv.HTTPRequestMethodOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HTTPRequestMethod(tt.method))
		})
	}
}

func TestHTTPRequestMethodAttributes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		want   []attribute.KeyValue
	}{
		{
			name:   "known method",
			method: "GET",
			want: []attribute.KeyValue{
				otelsemconv.HTTPRequestMethodGet,
			},
		},
		{
			name:   "unknown method",
			method: "CUSTOM",
			want: []attribute.KeyValue{
				otelsemconv.HTTPRequestMethodOther,
				otelsemconv.HTTPRequestMethodOriginal("CUSTOM"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, httpRequestMethodAttributes(tt.method))
		})
	}
}

func TestHTTPRequestBodySize(t *testing.T) {
	want := otelsemconv.HTTPRequestBodySize(7)

	assert.Equal(t, want, HTTPRequestBodySize(7))
}

func TestMethodAttributes(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		want       []attribute.KeyValue
	}{
		{
			name:       "/foo.bar/hello",
			fullMethod: "/foo.bar/hello",
			want: []attribute.KeyValue{
				otelsemconv.RPCMethod("foo.bar/hello"),
			},
		},
		{
			name:       "/foo.bar/hello/world",
			fullMethod: "/foo.bar/hello/world",
			want: []attribute.KeyValue{
				otelsemconv.RPCMethod("foo.bar/hello/world"),
			},
		},
		{
			name:       "/hello",
			fullMethod: "/hello",
			want: []attribute.KeyValue{
				otelsemconv.RPCMethod("_OTHER"),
				otelsemconv.RPCMethodOriginal("/hello"),
			},
		},
		{
			name:       "empty",
			fullMethod: "",
			want:       []attribute.KeyValue(nil),
		},
		{
			name:       "missing method",
			fullMethod: "/foo.bar/",
			want: []attribute.KeyValue{
				otelsemconv.RPCMethod("_OTHER"),
				otelsemconv.RPCMethodOriginal("/foo.bar/"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, methodAttributes(tt.fullMethod))
		})
	}
}

func TestPeer(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want []attribute.KeyValue
	}{
		{
			name: "nil addr",
			addr: ":8080",
			want: []attribute.KeyValue{
				otelsemconv.NetworkPeerAddress("127.0.0.1"),
				otelsemconv.NetworkPeerPort(8080),
			},
		},
		{
			name: "normal addr without port",
			addr: "192.168.0.1",
			want: []attribute.KeyValue(nil),
		},
		{
			name: "normal addr with port",
			addr: "192.168.0.1:8080",
			want: []attribute.KeyValue{
				otelsemconv.NetworkPeerAddress("192.168.0.1"),
				otelsemconv.NetworkPeerPort(8080),
			},
		},
		{
			name: "dns addr",
			addr: "foo:8080",
			want: []attribute.KeyValue{
				otelsemconv.NetworkPeerAddress("foo"),
				otelsemconv.NetworkPeerPort(8080),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Peer(tt.addr))
		})
	}
}

func TestRPCSystemName(t *testing.T) {
	tests := []struct {
		name string
		kind transport.Kind
		want attribute.KeyValue
	}{
		{
			name: "known grpc system",
			kind: transport.KindGRPC,
			want: otelsemconv.RPCSystemNameGRPC,
		},
		{
			name: "unknown system",
			kind: transport.KindHTTP,
			want: otelsemconv.RPCSystemNameKey.String(transport.KindHTTP.String()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RPCSystemName(tt.kind))
		})
	}
}

func TestRPCErrorAttributes(t *testing.T) {
	want := []attribute.KeyValue{
		otelsemconv.RPCResponseStatusCode("500"),
		otelsemconv.ErrorTypeKey.String("500"),
	}

	assert.Equal(t, want, RPCErrorAttributes(500))
}

func TestMessageSize(t *testing.T) {
	msg := wrapperspb.String("hello")
	var nilMsg *wrapperspb.StringValue

	tests := []struct {
		name string
		got  []attribute.KeyValue
		want []attribute.KeyValue
	}{
		{
			name: "send message size",
			got:  SendMessageSize(msg),
			want: []attribute.KeyValue{attribute.Key("send_msg.size").Int(proto.Size(msg))},
		},
		{
			name: "recv message size",
			got:  RecvMessageSize(msg),
			want: []attribute.KeyValue{attribute.Key("recv_msg.size").Int(proto.Size(msg))},
		},
		{
			name: "non protobuf message",
			got:  SendMessageSize("not-proto"),
			want: []attribute.KeyValue(nil),
		},
		{
			name: "nil protobuf message",
			got:  SendMessageSize(nilMsg),
			want: []attribute.KeyValue(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
