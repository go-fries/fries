package tracing

import (
	"net"
	"net/http"
	"testing"

	tracingpb "github.com/go-fries/fries/kratos/middleware/tracing/v3/internal/proto/tracing/v1"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/peer"
)

func TestSetServerSpan(t *testing.T) {
	ctx := t.Context()
	_, span := noop.NewTracerProvider().Tracer("Tracer").Start(ctx, "Spanname")

	// Handle without Transport context
	setServerSpan(ctx, span, nil)

	// Handle with proto message
	m := &tracingpb.HelloRequest{}
	setServerSpan(ctx, span, m)

	// Handle with metadata context
	ctx = metadata.NewServerContext(ctx, metadata.New())
	setServerSpan(ctx, span, m)

	// Handle with KindHTTP transport context
	mt := &mockTransport{
		kind: transport.KindHTTP,
	}
	mt.request, _ = http.NewRequest(http.MethodGet, "/endpoint", nil)
	ctx = transport.NewServerContext(ctx, mt)
	setServerSpan(ctx, span, m)

	// Handle with KindGRPC transport context
	mt.kind = transport.KindGRPC
	ctx = transport.NewServerContext(ctx, mt)
	ip, _ := net.ResolveIPAddr("ip", "1.1.1.1")
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: ip,
	})
	setServerSpan(ctx, span, m)
}

func TestSetClientSpan(t *testing.T) {
	ctx := t.Context()
	_, span := noop.NewTracerProvider().Tracer("Tracer").Start(ctx, "Spanname")

	// Handle without Transport context
	setClientSpan(ctx, span, nil)

	// Handle with proto message
	m := &tracingpb.HelloRequest{}
	setClientSpan(ctx, span, m)

	// Handle with metadata context
	ctx = metadata.NewClientContext(ctx, metadata.New())
	setClientSpan(ctx, span, m)

	// Handle with KindHTTP transport context
	mt := &mockTransport{
		kind: transport.KindHTTP,
	}
	mt.request, _ = http.NewRequest(http.MethodGet, "/endpoint", nil)
	mt.request.Host = "MyServer"
	ctx = transport.NewClientContext(ctx, mt)
	setClientSpan(ctx, span, m)

	// Handle with KindGRPC transport context
	mt.kind = transport.KindGRPC
	ctx = transport.NewClientContext(ctx, mt)
	ip, _ := net.ResolveIPAddr("ip", "1.1.1.1")
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: ip,
	})
	setClientSpan(ctx, span, m)

	// Handle without Host request
	ctx = transport.NewClientContext(ctx, mt)
	setClientSpan(ctx, span, m)
}
