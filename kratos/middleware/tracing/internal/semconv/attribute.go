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

func ClientAddress(address string) attribute.KeyValue {
	return otelsemconv.ClientAddress(address)
}

func ServerAddress(address string) attribute.KeyValue {
	return otelsemconv.ServerAddress(address)
}

func ServerPort(port int) attribute.KeyValue {
	return otelsemconv.ServerPort(port)
}

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

func HTTPRequestBodySize(size int) attribute.KeyValue {
	return otelsemconv.HTTPRequestBodySize(size)
}

func HTTPRoute(route string) attribute.KeyValue {
	return otelsemconv.HTTPRoute(route)
}

func NetworkPeerAddress(address string) attribute.KeyValue {
	return otelsemconv.NetworkPeerAddress(address)
}

func NetworkPeerPort(port int) attribute.KeyValue {
	return otelsemconv.NetworkPeerPort(port)
}

func RPCMethod(method string) attribute.KeyValue {
	return otelsemconv.RPCMethod(method)
}

func RPCMethodOriginal(method string) attribute.KeyValue {
	return otelsemconv.RPCMethodOriginal(method)
}

func RPCErrorAttributes(code int32) []attribute.KeyValue {
	statusCode := strconv.FormatInt(int64(code), 10)
	return []attribute.KeyValue{
		otelsemconv.RPCResponseStatusCode(statusCode),
		otelsemconv.ErrorTypeKey.String(statusCode),
	}
}

func RPCSystemName(kind transport.Kind) attribute.KeyValue {
	switch kind {
	case transport.KindGRPC:
		return otelsemconv.RPCSystemNameGRPC
	default:
		return otelsemconv.RPCSystemNameKey.String(kind.String())
	}
}

func ServicePeerName(name string) attribute.KeyValue {
	return otelsemconv.ServicePeerName(name)
}

func URLPath(path string) attribute.KeyValue {
	return otelsemconv.URLPath(path)
}

func URLFull(full string) attribute.KeyValue {
	return otelsemconv.URLFull(full)
}

func URLQuery(query string) attribute.KeyValue {
	return otelsemconv.URLQuery(query)
}

func URLScheme(scheme string) attribute.KeyValue {
	return otelsemconv.URLScheme(scheme)
}

func UserAgentOriginal(userAgent string) attribute.KeyValue {
	return otelsemconv.UserAgentOriginal(userAgent)
}

func SendMessageSize(m any) []attribute.KeyValue {
	return messageSize("send_msg.size", m)
}

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
