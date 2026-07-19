package health_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-fries/fries/health/v4"
)

func ExampleHandler() {
	readiness := health.New()
	readiness.Register("database", health.CheckFunc(func(context.Context) error {
		return nil
	}))

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	health.Handler(readiness).ServeHTTP(recorder, request)

	fmt.Println(recorder.Code)

	// Output:
	// 200
}
