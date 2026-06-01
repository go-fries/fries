package semconv

import (
	"context"
	"net"
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
				attrs = append(attrs, httpTransporter(ht)...)
				remote = ht.Request().Host
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
				attrs = append(attrs, httpTransporter(ht)...)
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

func httpTransporter(ht khttp.Transporter) []attribute.KeyValue {
	return []attribute.KeyValue{
		HTTPRequestMethod(ht.Request().Method),
		HTTPRoute(ht.PathTemplate()),
		URLPath(ht.Request().URL.Path),
		ClientAddress(ht.Request().RemoteAddr),
		UserAgentOriginal(ht.Request().UserAgent()),
	}
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
