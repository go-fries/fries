package tracing

import (
	"context"
	"net"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos-ecosystem/components/v2/hyperf/jet"
)

const instrumentation = "github.com/go-kratos-ecosystem/components/v2/hyperf/jet/middleware/tracing"

type options struct {
	mp    propagation.TextMapPropagator
	tp    trace.TracerProvider
	attrs []attribute.KeyValue
}

type Option func(*options)

func WithPropagator(mp propagation.TextMapPropagator) Option {
	return func(o *options) {
		o.mp = mp
	}
}

func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(o *options) {
		o.tp = tp
	}
}

func WithAttributes(attrs ...attribute.KeyValue) Option {
	return func(o *options) {
		o.attrs = append(o.attrs, attrs...)
	}
}

func newOptions(opts ...Option) options {
	o := options{
		mp: otel.GetTextMapPropagator(),
		tp: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func New(opts ...Option) jet.Middleware {
	o := newOptions(opts...)

	tracer := o.tp.Tracer(instrumentation)
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, service, method string, request any) (response any, err error) {
			ctx, span := tracer.Start(ctx, "jet."+service+"/"+method,
				trace.WithSpanKind(trace.SpanKindClient),
			)
			defer span.End()

			attrs := []attribute.KeyValue{
				semconv.RPCService(service),
				semconv.RPCMethod(method),
				// semconv.RPCJsonrpcErrorCode(0),     // todo
				// semconv.RPCJsonrpcErrorMessage(""), // todo
				// semconv.RPCJsonrpcRequestID(""),    // todo
				// semconv.ServerAddress(""),          // todo
				// semconv.ServerPort(0),              // todo
				// semconv.NetworkPeerAddress(""),     // todo
				// semconv.NetworkPeerPort(0),         // todo
			}
			attrs = append(attrs, formatterAttributes(ctx)...)
			attrs = append(attrs, transportAttributes(ctx)...)
			attrs = append(attrs, o.attrs...)

			span.SetAttributes(attrs...)

			response, err = next(ctx, service, method, request)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.RecordError(err)
			}

			return
		}
	}
}

func formatterAttributes(ctx context.Context) []attribute.KeyValue {
	client, ok := jet.ClientFromContext(ctx)
	if !ok {
		return []attribute.KeyValue{}
	}

	switch formatter := client.GetFormatter(); formatter.Kind() {
	case jet.FormatterKindJSONRPC:
		return []attribute.KeyValue{
			semconv.RPCSystemKey.String("jsonrpc"),
			semconv.RPCJsonrpcVersion(jet.JSONRPCVersion),
		}
	default:
		return []attribute.KeyValue{}
	}
}

func transportAttributes(ctx context.Context) []attribute.KeyValue {
	client, ok := jet.ClientFromContext(ctx)
	if !ok {
		return []attribute.KeyValue{}
	}

	switch transporter := client.GetTransporter().(type) {
	case *jet.HTTPTransporter:
		var attrs []attribute.KeyValue
		if host, port, err := net.SplitHostPort(transporter.Addr); err == nil {
			attrs = append(attrs, semconv.ServerAddress(host))
			if p, err := strconv.Atoi(port); err == nil {
				attrs = append(attrs, semconv.ServerPort(p))
			}
		}
		return attrs
	default:
		return []attribute.KeyValue{}
	}
}
