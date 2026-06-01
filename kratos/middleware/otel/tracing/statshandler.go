package tracing

import (
	"context"

	"github.com/go-fries/fries/kratos/middleware/otel/tracing/v3/internal/semconv"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

// ClientHandler is a [stats.Handler] that adds peer attributes to client spans.
type ClientHandler struct{}

// HandleConn satisfies [stats.Handler].
func (c *ClientHandler) HandleConn(context.Context, stats.ConnStats) {}

// TagConn satisfies [stats.Handler].
func (c *ClientHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleRPC implements per-RPC tracing and stats instrumentation.
func (c *ClientHandler) HandleRPC(ctx context.Context, rs stats.RPCStats) {
	if _, ok := rs.(*stats.OutHeader); !ok {
		return
	}
	p, ok := peer.FromContext(ctx)
	if !ok {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		span.SetAttributes(semconv.Peer(p.Addr.String())...)
	}
}

// TagRPC implements per-RPC context management for [stats.Handler].
func (c *ClientHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	return ctx
}
