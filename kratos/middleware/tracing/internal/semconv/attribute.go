package semconv

import (
	"net/http"
	"reflect"
	"strconv"

	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"google.golang.org/protobuf/proto"
)

// ClientAddress returns the client.address semantic-convention attribute.
func ClientAddress(address string) attribute.KeyValue {
	return otelsemconv.ClientAddress(address)
}

// ServerAddress returns the server.address semantic-convention attribute.
func ServerAddress(address string) attribute.KeyValue {
	return otelsemconv.ServerAddress(address)
}

// ServerPort returns the server.port semantic-convention attribute.
func ServerPort(port int) attribute.KeyValue {
	return otelsemconv.ServerPort(port)
}

// HTTPRequestMethod returns the HTTP request method semantic-convention attribute.
func HTTPRequestMethod(method string) attribute.KeyValue {
	switch method {
	case http.MethodConnect:
		return otelsemconv.HTTPRequestMethodConnect
	case http.MethodDelete:
		return otelsemconv.HTTPRequestMethodDelete
	case http.MethodGet:
		return otelsemconv.HTTPRequestMethodGet
	case http.MethodHead:
		return otelsemconv.HTTPRequestMethodHead
	case http.MethodOptions:
		return otelsemconv.HTTPRequestMethodOptions
	case http.MethodPatch:
		return otelsemconv.HTTPRequestMethodPatch
	case http.MethodPost:
		return otelsemconv.HTTPRequestMethodPost
	case http.MethodPut:
		return otelsemconv.HTTPRequestMethodPut
	case http.MethodTrace:
		return otelsemconv.HTTPRequestMethodTrace
	case "QUERY":
		return otelsemconv.HTTPRequestMethodQuery
	case "_OTHER":
		return otelsemconv.HTTPRequestMethodOther
	default:
		return otelsemconv.HTTPRequestMethodKey.String(method)
	}
}

// HTTPRequestBodySize returns the HTTP request body size semantic-convention attribute.
func HTTPRequestBodySize(size int) attribute.KeyValue {
	return otelsemconv.HTTPRequestBodySize(size)
}

// HTTPRoute returns the HTTP route semantic-convention attribute.
func HTTPRoute(route string) attribute.KeyValue {
	return otelsemconv.HTTPRoute(route)
}

// NetworkPeerAddress returns the network peer address semantic-convention attribute.
func NetworkPeerAddress(address string) attribute.KeyValue {
	return otelsemconv.NetworkPeerAddress(address)
}

// NetworkPeerPort returns the network peer port semantic-convention attribute.
func NetworkPeerPort(port int) attribute.KeyValue {
	return otelsemconv.NetworkPeerPort(port)
}

// RPCMethod returns the RPC method semantic-convention attribute.
func RPCMethod(method string) attribute.KeyValue {
	return otelsemconv.RPCMethod(method)
}

// RPCMethodOriginal returns the original RPC method semantic-convention attribute.
func RPCMethodOriginal(method string) attribute.KeyValue {
	return otelsemconv.RPCMethodOriginal(method)
}

// RPCErrorAttributes returns RPC error status attributes for code.
func RPCErrorAttributes(code int32) []attribute.KeyValue {
	statusCode := strconv.FormatInt(int64(code), 10)
	return []attribute.KeyValue{
		otelsemconv.RPCResponseStatusCode(statusCode),
		otelsemconv.ErrorTypeKey.String(statusCode),
	}
}

// RPCSystemName returns the RPC system name semantic-convention attribute.
func RPCSystemName(kind transport.Kind) attribute.KeyValue {
	switch kind {
	case transport.KindGRPC:
		return otelsemconv.RPCSystemNameGRPC
	default:
		return otelsemconv.RPCSystemNameKey.String(kind.String())
	}
}

// ServicePeerName returns the service peer name semantic-convention attribute.
func ServicePeerName(name string) attribute.KeyValue {
	return otelsemconv.ServicePeerName(name)
}

// URLPath returns the URL path semantic-convention attribute.
func URLPath(path string) attribute.KeyValue {
	return otelsemconv.URLPath(path)
}

// URLFull returns the full URL semantic-convention attribute.
func URLFull(full string) attribute.KeyValue {
	return otelsemconv.URLFull(full)
}

// URLQuery returns the URL query semantic-convention attribute.
func URLQuery(query string) attribute.KeyValue {
	return otelsemconv.URLQuery(query)
}

// URLScheme returns the URL scheme semantic-convention attribute.
func URLScheme(scheme string) attribute.KeyValue {
	return otelsemconv.URLScheme(scheme)
}

// UserAgentOriginal returns the original user agent semantic-convention attribute.
func UserAgentOriginal(userAgent string) attribute.KeyValue {
	return otelsemconv.UserAgentOriginal(userAgent)
}

// SendMessageSize returns the outgoing protobuf message size attribute.
func SendMessageSize(m any) []attribute.KeyValue {
	return messageSize("send_msg.size", m)
}

// RecvMessageSize returns the incoming protobuf message size attribute.
func RecvMessageSize(m any) []attribute.KeyValue {
	return messageSize("recv_msg.size", m)
}

func messageSize(key string, m any) []attribute.KeyValue {
	if p, ok := m.(proto.Message); ok {
		if isNilProtoMessage(p) {
			return nil
		}
		return []attribute.KeyValue{attribute.Key(key).Int(proto.Size(p))}
	}
	return nil
}

func isNilProtoMessage(m proto.Message) bool {
	if m == nil {
		return true
	}
	v := reflect.ValueOf(m)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
