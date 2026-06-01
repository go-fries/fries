package tracing

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

func setClientSpan(ctx context.Context, span trace.Span, m any) {
	var (
		attrs     []attribute.KeyValue
		remote    string
		operation string
	)
	tr, ok := transport.FromClientContext(ctx)
	if ok {
		operation = tr.Operation()
		switch tr.Kind() {
		case transport.KindHTTP:
			if ht, ok := tr.(http.Transporter); ok {
				attrs = append(
					attrs,
					semconv.HTTPRequestMethodKey.String(ht.Request().Method),
					semconv.HTTPRoute(ht.PathTemplate()),
					semconv.URLPath(ht.Request().URL.Path),
					semconv.ClientAddress(ht.Request().RemoteAddr),
					semconv.UserAgentOriginal(ht.Request().UserAgent()),
				)
				remote = ht.Request().Host
			}
		case transport.KindGRPC:
			remote, _ = parseTarget(tr.Endpoint())
		}
		attrs = append(attrs, rpcSystemName(tr.Kind()))
	}
	_, mAttrs := parseFullMethod(operation)
	attrs = append(attrs, mAttrs...)
	if remote != "" {
		attrs = append(attrs, peerAttr(remote)...)
	}
	if p, ok := m.(proto.Message); ok {
		attrs = append(attrs, attribute.Key("send_msg.size").Int(proto.Size(p)))
	}

	span.SetAttributes(attrs...)
}

func setServerSpan(ctx context.Context, span trace.Span, m any) {
	var (
		attrs     []attribute.KeyValue
		remote    string
		operation string
	)
	tr, ok := transport.FromServerContext(ctx)
	if ok {
		operation = tr.Operation()
		switch tr.Kind() {
		case transport.KindHTTP:
			if ht, ok := tr.(http.Transporter); ok {
				attrs = append(
					attrs,
					semconv.HTTPRequestMethodKey.String(ht.Request().Method),
					semconv.HTTPRoute(ht.PathTemplate()),
					semconv.URLPath(ht.Request().URL.Path),
					semconv.ClientAddress(ht.Request().RemoteAddr),
					semconv.UserAgentOriginal(ht.Request().UserAgent()),
				)
				remote = ht.Request().RemoteAddr
			}
		case transport.KindGRPC:
			if p, ok := peer.FromContext(ctx); ok {
				remote = p.Addr.String()
			}
		}
		attrs = append(attrs, rpcSystemName(tr.Kind()))
	}
	_, mAttrs := parseFullMethod(operation)
	attrs = append(attrs, mAttrs...)
	attrs = append(attrs, peerAttr(remote)...)
	if p, ok := m.(proto.Message); ok {
		attrs = append(attrs, attribute.Key("recv_msg.size").Int(proto.Size(p)))
	}
	if md, ok := metadata.FromServerContext(ctx); ok {
		attrs = append(attrs, semconv.ServicePeerName(md.Get(serviceHeader)))
	}

	span.SetAttributes(attrs...)
}

func rpcSystemName(kind transport.Kind) attribute.KeyValue {
	switch kind {
	case transport.KindGRPC:
		return semconv.RPCSystemNameGRPC
	default:
		return semconv.RPCSystemNameKey.String(kind.String())
	}
}

// parseFullMethod returns a span name following the OpenTelemetry semantic
// conventions as well as all applicable span attribute.KeyValue attributes based
// on a gRPC's FullMethod.
func parseFullMethod(fullMethod string) (string, []attribute.KeyValue) {
	name := strings.TrimLeft(fullMethod, "/")
	parts := strings.SplitN(name, "/", 2) //nolint:mnd
	if len(parts) != 2 {                  //nolint:mnd
		// Invalid format, does not follow `/package.service/method`.
		return name, []attribute.KeyValue{attribute.Key("rpc.operation").String(fullMethod)}
	}

	var attrs []attribute.KeyValue
	if service := parts[0]; service != "" {
		if method := parts[1]; method != "" {
			attrs = append(attrs, semconv.RPCMethod(service+"/"+method))
		}
	} else if method := parts[1]; method != "" {
		attrs = append(attrs, semconv.RPCMethod(method))
	}
	return name, attrs
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) (attrs []attribute.KeyValue) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return attrs
	}

	if host == "" {
		host = "127.0.0.1"
	}
	attrs = append(
		attrs,
		semconv.NetworkPeerAddress(host),
	)

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return attrs
	}
	attrs = append(
		attrs,
		semconv.NetworkPeerPort(portInt),
	)

	return attrs
}

func parseTarget(endpoint string) (address string, err error) {
	var u *url.URL
	u, err = url.Parse(endpoint)
	if err != nil {
		if u, err = url.Parse("http://" + endpoint); err != nil {
			return "", err
		}
		return u.Host, nil
	}
	if len(u.Path) > 1 {
		return u.Path[1:], nil
	}
	return endpoint, nil
}
