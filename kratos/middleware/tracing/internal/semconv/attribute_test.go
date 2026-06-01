package semconv

import (
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
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
			want:   otelsemconv.HTTPRequestMethodKey.String("CUSTOM"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTTPRequestMethod(tt.method); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HTTPRequestMethod() = %v, want %v", got, tt.want)
			}
		})
	}
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
			if got := methodAttributes(tt.fullMethod); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("methodAttributes() = %v, want %v", got, tt.want)
			}
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
			if got := Peer(tt.addr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Peer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		wantAddress string
		wantErr     bool
	}{
		{
			name:        "http",
			endpoint:    "http://foo.bar:8080",
			wantAddress: "http://foo.bar:8080",
		},
		{
			name:        "http",
			endpoint:    "http://127.0.0.1:8080",
			wantAddress: "http://127.0.0.1:8080",
		},
		{
			name:        "without protocol",
			endpoint:    "foo.bar:8080",
			wantAddress: "foo.bar:8080",
		},
		{
			name:        "grpc",
			endpoint:    "grpc://foo.bar",
			wantAddress: "grpc://foo.bar",
		},
		{
			name:        "with path",
			endpoint:    "/foo",
			wantAddress: "foo",
		},
		{
			name:        "with path",
			endpoint:    "http://127.0.0.1/hello",
			wantAddress: "hello",
		},
		{
			name:     "empty",
			endpoint: "%%",
			wantErr:  true,
		},
		{
			name:     "invalid path",
			endpoint: "//%2F/#%2Fanother",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddress, err := parseTarget(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAddress != tt.wantAddress {
				t.Errorf("parseTarget() = %v, want %v", gotAddress, tt.wantAddress)
			}
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
			if got := RPCSystemName(tt.kind); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RPCSystemName() = %v, want %v", got, tt.want)
			}
		})
	}
}
