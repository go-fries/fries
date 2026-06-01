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
