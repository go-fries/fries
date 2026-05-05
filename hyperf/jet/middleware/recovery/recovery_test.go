package recovery

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	testService := "service"
	testMethod := "method"
	testRequest := "request"
	testError := "error"

	recovery := New(
		Handler(func(_ context.Context, service, method string, request, err any) error {
			assert.Equal(t, testService, service)
			assert.Equal(t, testMethod, method)
			assert.Equal(t, testRequest, request)
			assert.Equal(t, testError, err)
			return fmt.Errorf("method: %s, request: %v, error: %v", method, request, err)
		}),
	)

	response, err := recovery(func(context.Context, string, string, any) (response any, err error) {
		panic(testError)
	})(t.Context(), testService, testMethod, testRequest)
	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestRecovery_DefaultHandler(t *testing.T) {
	recovery := New()

	response, err := recovery(func(context.Context, string, string, any) (response any, err error) {
		panic("error")
	})(t.Context(), "service", "method", "request")
	assert.Error(t, err)
	assert.Nil(t, response)
}
