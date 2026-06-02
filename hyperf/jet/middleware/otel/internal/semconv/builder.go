package semconv

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-fries/fries/hyperf/jet/v3"
	"go.opentelemetry.io/otel/attribute"
)

// Builder builds OpenTelemetry semantic-convention attributes for Jet clients.
type Builder struct{}

// NewBuilder returns a Builder.
func NewBuilder() Builder {
	return Builder{}
}

// Client returns attributes for a client span.
func (b Builder) Client(ctx context.Context, service, method string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 6)
	attrs = append(attrs, RPCMethod(b.Operation(ctx, service, method)))

	client, ok := jet.ClientFromContext(ctx)
	if !ok {
		return attrs
	}

	attrs = append(attrs, formatterAttributes(client.GetFormatter())...)
	attrs = append(attrs, transporterAttributes(client.GetTransporter())...)

	return attrs
}

// Operation returns the Jet RPC operation name for service and method.
func (Builder) Operation(ctx context.Context, service, method string) string {
	client, ok := jet.ClientFromContext(ctx)
	if !ok {
		return methodName(service, method)
	}

	pathGenerator := client.GetPathGenerator()
	if pathGenerator == nil {
		return methodName(service, method)
	}

	return pathGenerator.Generate(service, method)
}

func formatterAttributes(formatter jet.Formatter) []attribute.KeyValue {
	if formatter == nil {
		return nil
	}

	switch formatter.Kind() {
	case jet.FormatterKindJSONRPC:
		return []attribute.KeyValue{
			RPCSystemNameJSONRPC(),
			JSONRPCProtocolVersion(jet.JSONRPCVersion),
		}
	default:
		return nil
	}
}

func transporterAttributes(transporter jet.Transporter) []attribute.KeyValue {
	switch t := transporter.(type) {
	case *jet.HTTPTransporter:
		return httpTransporterAttributes(t)
	default:
		return nil
	}
}

func httpTransporterAttributes(t *jet.HTTPTransporter) []attribute.KeyValue {
	if t == nil || t.Addr == "" {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, 2)
	if address, port, ok := serverAddress(t.Addr); ok {
		attrs = append(attrs, ServerAddress(address))
		if port > 0 {
			attrs = append(attrs, ServerPort(port))
		}
	}

	return attrs
}

func serverAddress(addr string) (address string, port int, ok bool) {
	if u, err := url.Parse(addr); err == nil && u.Host != "" {
		return splitHostPort(u.Host, defaultPort(u.Scheme))
	}

	return splitHostPort(addr, 0)
}

func splitHostPort(hostport string, fallbackPort int) (address string, port int, ok bool) {
	host, portValue, err := net.SplitHostPort(hostport)
	if err == nil {
		if p, err := strconv.Atoi(portValue); err == nil {
			port = p
		}
		return host, port, host != ""
	}

	host = hostport
	if h, p, ok := strings.Cut(hostport, ":"); ok {
		host = h
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}
	if port == 0 {
		port = fallbackPort
	}

	return host, port, host != ""
}

func methodName(service, method string) string {
	switch {
	case service == "":
		return method
	case method == "":
		return service
	default:
		return service + "/" + method
	}
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
