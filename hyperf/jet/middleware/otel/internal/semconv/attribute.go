package semconv

import (
	"errors"
	"strconv"

	"github.com/go-fries/fries/hyperf/jet/v3"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// HTTPResponseStatusCode returns the HTTP response status code attribute.
func HTTPResponseStatusCode(code int) attribute.KeyValue {
	return otelsemconv.HTTPResponseStatusCode(code)
}

// JSONRPCProtocolVersion returns the JSON-RPC protocol version attribute.
func JSONRPCProtocolVersion(version string) attribute.KeyValue {
	return otelsemconv.JSONRPCProtocolVersion(version)
}

// RPCErrorAttributes returns RPC error status attributes for code.
func RPCErrorAttributes(code int) []attribute.KeyValue {
	statusCode := strconv.Itoa(code)
	return []attribute.KeyValue{
		otelsemconv.RPCResponseStatusCode(statusCode),
		otelsemconv.ErrorTypeKey.String(statusCode),
	}
}

// RPCMethod returns the RPC method attribute.
func RPCMethod(method string) attribute.KeyValue {
	return otelsemconv.RPCMethod(method)
}

// RPCSystemNameJSONRPC returns the JSON-RPC system name attribute.
func RPCSystemNameJSONRPC() attribute.KeyValue {
	return otelsemconv.RPCSystemNameJSONRPC
}

// ServerAddress returns the server address attribute.
func ServerAddress(address string) attribute.KeyValue {
	return otelsemconv.ServerAddress(address)
}

// ServerPort returns the server port attribute.
func ServerPort(port int) attribute.KeyValue {
	return otelsemconv.ServerPort(port)
}

// ErrorAttributes returns semantic-convention attributes derived from err.
func ErrorAttributes(err error) []attribute.KeyValue {
	var rpcErr *jet.RPCResponseError
	if errors.As(err, &rpcErr) {
		return RPCErrorAttributes(rpcErr.Code)
	}

	var httpErr *jet.HTTPTransporterServerError
	if errors.As(err, &httpErr) {
		statusCode := strconv.Itoa(httpErr.StatusCode)
		return []attribute.KeyValue{
			HTTPResponseStatusCode(httpErr.StatusCode),
			otelsemconv.ErrorTypeKey.String(statusCode),
		}
	}

	return nil
}
