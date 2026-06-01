package semconv

import (
	"net/http"

	"github.com/go-kratos/kratos/v2/transport"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func ClientAddress(address string) attribute.KeyValue {
	return otelsemconv.ClientAddress(address)
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

func RPCOperation(operation string) attribute.KeyValue {
	return attribute.Key("rpc.operation").String(operation)
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

func UserAgentOriginal(userAgent string) attribute.KeyValue {
	return otelsemconv.UserAgentOriginal(userAgent)
}
