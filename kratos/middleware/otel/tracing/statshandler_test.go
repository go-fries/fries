package tracing

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

type ctxKey string

const testKey ctxKey = "MY_TEST_KEY"

func TestClient_HandleConn(t *testing.T) {
	(&ClientHandler{}).HandleConn(t.Context(), nil)
}

func TestClient_TagConn(t *testing.T) {
	client := &ClientHandler{}
	ctx := context.WithValue(t.Context(), testKey, 123)

	assert.Equal(t, 123, client.TagConn(ctx, nil).Value(testKey))
}

func TestClient_TagRPC(t *testing.T) {
	client := &ClientHandler{}
	ctx := context.WithValue(t.Context(), testKey, 123)

	assert.Equal(t, 123, client.TagRPC(ctx, nil).Value(testKey))
}

type mockSpan struct {
	trace.Span
	mockSpanCtx *trace.SpanContext
}

func (m *mockSpan) SpanContext() trace.SpanContext {
	return *m.mockSpanCtx
}

func TestClient_HandleRPC(t *testing.T) {
	client := &ClientHandler{}
	ctx := t.Context()
	rs := stats.OutHeader{}

	// Handle stats.RPCStats is not type of stats.OutHeader case
	client.HandleRPC(t.Context(), nil)

	// Handle context doesn't have the peerkey filled with a Peer instance
	client.HandleRPC(ctx, &rs)

	// Handle context with the peerkey filled with a Peer instance
	ip, err := net.ResolveIPAddr("ip", "1.1.1.1")
	require.NoError(t, err)
	ctx = peer.NewContext(ctx, &peer.Peer{
		Addr: ip,
	})
	client.HandleRPC(ctx, &rs)

	// Handle context with Span
	_, span := noop.NewTracerProvider().Tracer("Tracer").Start(ctx, "Spanname")
	spanCtx := trace.SpanContext{}
	spanID := [8]byte{12, 12, 12, 12, 12, 12, 12, 12}
	traceID := [16]byte{12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12, 12}
	spanCtx = spanCtx.WithTraceID(traceID)
	spanCtx = spanCtx.WithSpanID(spanID)
	mSpan := mockSpan{
		Span:        span,
		mockSpanCtx: &spanCtx,
	}
	ctx = trace.ContextWithSpan(ctx, &mSpan)
	client.HandleRPC(ctx, &rs)
}
