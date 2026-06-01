package semconv

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

type Builder struct {
	serviceHeader string
}

func NewBuilder(serviceHeader string) Builder {
	return Builder{serviceHeader: serviceHeader}
}

func (b Builder) Client(ctx context.Context, m any) []attribute.KeyValue {
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
			if ht, ok := tr.(khttp.Transporter); ok {
				attrs = append(attrs, httpClientTransporter(ht)...)
			}
		case transport.KindGRPC:
			remote, _ = parseTarget(tr.Endpoint())
		}
		attrs = append(attrs, RPCSystemName(tr.Kind()))
	}

	attrs = append(attrs, methodAttributes(operation)...)
	if remote != "" {
		attrs = append(attrs, Peer(remote)...)
	}
	attrs = append(attrs, messageSize("send_msg.size", m)...)

	return attrs
}

func (b Builder) Server(ctx context.Context, m any) []attribute.KeyValue {
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
			if ht, ok := tr.(khttp.Transporter); ok {
				attrs = append(attrs, httpServerTransporter(ht)...)
				remote = ht.Request().RemoteAddr
			}
		case transport.KindGRPC:
			if p, ok := peer.FromContext(ctx); ok {
				remote = p.Addr.String()
			}
		}
		attrs = append(attrs, RPCSystemName(tr.Kind()))
	}

	attrs = append(attrs, methodAttributes(operation)...)
	attrs = append(attrs, Peer(remote)...)
	attrs = append(attrs, messageSize("recv_msg.size", m)...)
	if md, ok := metadata.FromServerContext(ctx); ok {
		attrs = append(attrs, ServicePeerName(md.Get(b.serviceHeader)))
	}

	return attrs
}

func httpClientTransporter(ht khttp.Transporter) []attribute.KeyValue {
	req := ht.Request()
	attrs := []attribute.KeyValue{
		HTTPRequestMethod(ht.Request().Method),
	}
	if req.URL != nil && req.URL.String() != "" {
		attrs = append(attrs, URLFull(req.URL.String()))
	}
	if address, port, ok := serverAddress(req); ok {
		attrs = append(attrs, ServerAddress(address))
		if port > 0 {
			attrs = append(attrs, ServerPort(port))
		}
	}
	if userAgent := req.UserAgent(); userAgent != "" {
		attrs = append(attrs, UserAgentOriginal(userAgent))
	}
	return attrs
}

func httpServerTransporter(ht khttp.Transporter) []attribute.KeyValue {
	req := ht.Request()
	attrs := []attribute.KeyValue{
		HTTPRequestMethod(req.Method),
		URLPath(req.URL.Path),
		URLScheme(requestScheme(req)),
	}
	if query := req.URL.RawQuery; query != "" {
		attrs = append(attrs, URLQuery(query))
	}
	if route := ht.PathTemplate(); route != "" {
		attrs = append(attrs, HTTPRoute(route))
	}
	if address, port, ok := serverAddress(req); ok {
		attrs = append(attrs, ServerAddress(address))
		if port > 0 {
			attrs = append(attrs, ServerPort(port))
		}
	}
	if address, ok := remoteAddress(req.RemoteAddr); ok {
		attrs = append(attrs, ClientAddress(address))
	}
	if userAgent := req.UserAgent(); userAgent != "" {
		attrs = append(attrs, UserAgentOriginal(userAgent))
	}
	return attrs
}

func messageSize(key string, m any) []attribute.KeyValue {
	if p, ok := m.(proto.Message); ok {
		return []attribute.KeyValue{attribute.Key(key).Int(proto.Size(p))}
	}
	return nil
}

// methodAttributes returns attributes about the gRPC full method.
func methodAttributes(fullMethod string) []attribute.KeyValue {
	name := strings.TrimLeft(fullMethod, "/")
	parts := strings.SplitN(name, "/", 2) //nolint:mnd
	if len(parts) != 2 {                  //nolint:mnd
		// Invalid format, does not follow `/package.service/method`.
		return []attribute.KeyValue{RPCOperation(fullMethod)}
	}

	var attrs []attribute.KeyValue
	if service := parts[0]; service != "" {
		if method := parts[1]; method != "" {
			attrs = append(attrs, RPCMethod(service+"/"+method))
		}
	} else if method := parts[1]; method != "" {
		attrs = append(attrs, RPCMethod(method))
	}
	return attrs
}

// Peer returns attributes about the peer address.
func Peer(addr string) (attrs []attribute.KeyValue) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return attrs
	}

	if host == "" {
		host = "127.0.0.1"
	}
	attrs = append(
		attrs,
		NetworkPeerAddress(host),
	)

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return attrs
	}
	attrs = append(
		attrs,
		NetworkPeerPort(portInt),
	)

	return attrs
}

func requestScheme(req *http.Request) string {
	if req.URL != nil && req.URL.Scheme != "" {
		return req.URL.Scheme
	}
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func serverAddress(req *http.Request) (string, int, bool) {
	if req.URL != nil && req.URL.Host != "" {
		return splitHostPort(req.URL.Host, defaultPort(req.URL.Scheme))
	}
	if req.Host != "" {
		return splitHostPort(req.Host, defaultPort(requestScheme(req)))
	}
	return "", 0, false
}

func remoteAddress(addr string) (string, bool) {
	host, _, err := net.SplitHostPort(addr)
	if err == nil {
		return host, host != ""
	}
	if addr == "" {
		return "", false
	}
	return addr, true
}

func splitHostPort(addr string, fallbackPort int) (string, int, bool) {
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return host, 0, host != ""
		}
		return host, portInt, host != ""
	}
	if host := strings.Trim(addr, "[]"); host != "" {
		return host, fallbackPort, true
	}
	return "", 0, false
}

func defaultPort(scheme string) int {
	switch scheme {
	case "http":
		return 80
	case "https":
		return 443
	default:
		return 0
	}
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
