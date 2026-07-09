package semconv

import (
	"errors"
	"testing"

	"github.com/go-fries/fries/hyperf/jet/v4"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

func TestErrorAttributesForRPCResponseError(t *testing.T) {
	err := &jet.RPCResponseError{
		Code:    -32603,
		Message: "internal error",
		Err:     errors.New("boom"),
	}

	got := ErrorAttributes(err)

	assert.Equal(t, []attribute.KeyValue{
		otelsemconv.RPCResponseStatusCode("-32603"),
		otelsemconv.ErrorTypeKey.String("-32603"),
	}, got)
}

func TestErrorAttributesForHTTPTransporterServerError(t *testing.T) {
	err := &jet.HTTPTransporterServerError{
		StatusCode: 503,
		Message:    "Service Unavailable",
		Err:        errors.New("boom"),
	}

	got := ErrorAttributes(err)

	assert.Equal(t, []attribute.KeyValue{
		otelsemconv.HTTPResponseStatusCode(503),
		otelsemconv.ErrorTypeKey.String("503"),
	}, got)
}

func TestErrorAttributesForUnknownError(t *testing.T) {
	got := ErrorAttributes(errors.New("boom"))

	assert.Empty(t, got)
}
