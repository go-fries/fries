package tracing

import (
	"context"
	"net/http"
	"testing"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var _ transport.Transporter = (*mockTransport)(nil)

type headerCarrier http.Header

// Get returns the value associated with the passed key.
func (hc headerCarrier) Get(key string) string {
	return http.Header(hc).Get(key)
}

// Set stores the key-value pair.
func (hc headerCarrier) Set(key, value string) {
	http.Header(hc).Set(key, value)
}

// Add value to the key-value pair.
func (hc headerCarrier) Add(key, value string) {
	http.Header(hc).Add(key, value)
}

// Keys lists the keys stored in this carrier.
func (hc headerCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range http.Header(hc) {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice value associated with the passed key.
func (hc headerCarrier) Values(key string) []string {
	return http.Header(hc).Values(key)
}

type mockTransport struct {
	kind      transport.Kind
	endpoint  string
	operation string
	header    headerCarrier
	request   *http.Request
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
func (tr *mockTransport) PathTemplate() string { return "" }

func traceIDs(ctx context.Context) (string, string) {
	spanContext := trace.SpanContextFromContext(ctx)
	var spanID, traceID string
	if spanContext.HasSpanID() {
		spanID = spanContext.SpanID().String()
	}
	if spanContext.HasTraceID() {
		traceID = spanContext.TraceID().String()
	}
	return spanID, traceID
}

func TestTracer(t *testing.T) {
	carrier := headerCarrier{}
	tp := tracesdk.NewTracerProvider(tracesdk.WithSampler(tracesdk.TraceIDRatioBased(0)))

	// caller use Inject
	cliTracer := newTracer(
		trace.SpanKindClient,
		WithTracerProvider(tp),
		WithPropagator(
			propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}),
		),
	)

	ts := &mockTransport{kind: transport.KindHTTP, header: carrier}

	ctx, aboveSpan := cliTracer.start(
		transport.NewClientContext(t.Context(), ts),
		ts.Operation(), ts.RequestHeader(),
	)
	defer cliTracer.end(ctx, aboveSpan, nil, nil)

	// server use Extract fetch traceInfo from carrier
	svrTracer := newTracer(trace.SpanKindServer,
		WithPropagator(
			propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{}),
		))
	ts = &mockTransport{kind: transport.KindHTTP, header: carrier}

	ctx, span := svrTracer.start(transport.NewServerContext(ctx, ts), ts.Operation(), ts.RequestHeader())
	defer svrTracer.end(ctx, span, nil, nil)

	assert.Equal(t, aboveSpan.SpanContext().TraceID(), span.SpanContext().TraceID())

	v, ok := transport.FromClientContext(ctx)
	require.True(t, ok)
	assert.NotEmpty(t, v.RequestHeader().Keys())
}

func TestServer(t *testing.T) {
	tr := &mockTransport{
		kind:      transport.KindHTTP,
		endpoint:  "server:2233",
		operation: "/test.server/hello",
		header:    headerCarrier{},
	}

	tracer := newTracer(
		trace.SpanKindClient,
		WithTracerProvider(tracesdk.NewTracerProvider()),
	)

	var (
		childSpanID  string
		childTraceID string
	)
	next := func(ctx context.Context, req any) (any, error) {
		childSpanID, childTraceID = traceIDs(ctx)
		return req.(string) + "https://go-kratos.dev", nil
	}

	var ctx context.Context
	ctx, span := tracer.start(
		transport.NewServerContext(t.Context(), tr),
		tr.Operation(),
		tr.RequestHeader(),
	)

	_, err := Server(
		WithTracerProvider(tracesdk.NewTracerProvider()),
		WithPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})),
	)(next)(ctx, "test server: ")

	span.End()
	assert.NoError(t, err)
	assert.NotEmpty(t, childSpanID)
	assert.NotEqual(t, span.SpanContext().SpanID().String(), childSpanID)
	assert.Equal(t, span.SpanContext().TraceID().String(), childTraceID)

	_, err = Server(
		WithTracerProvider(tracesdk.NewTracerProvider()),
		WithPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})),
	)(next)(t.Context(), "test server: ")
	assert.NoError(t, err)
	assert.Empty(t, childSpanID)
	assert.Empty(t, childTraceID)
}

func TestClient(t *testing.T) {
	tr := &mockTransport{
		kind:      transport.KindHTTP,
		endpoint:  "server:2233",
		operation: "/test.server/hello",
		header:    headerCarrier{},
	}

	tracer := newTracer(
		trace.SpanKindClient,
		WithTracerProvider(tracesdk.NewTracerProvider()),
	)

	var (
		childSpanID  string
		childTraceID string
	)
	next := func(ctx context.Context, req any) (any, error) {
		childSpanID, childTraceID = traceIDs(ctx)
		return req.(string) + "https://go-kratos.dev", nil
	}

	var ctx context.Context
	ctx, span := tracer.start(
		transport.NewClientContext(t.Context(), tr),
		tr.Operation(),
		tr.RequestHeader(),
	)

	_, err := Client(
		WithTracerProvider(tracesdk.NewTracerProvider()),
		WithPropagator(propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})),
	)(next)(ctx, "test client: ")

	span.End()
	assert.NoError(t, err)
	assert.NotEmpty(t, childSpanID)
	assert.NotEqual(t, span.SpanContext().SpanID().String(), childSpanID)
	assert.Equal(t, span.SpanContext().TraceID().String(), childTraceID)
}
