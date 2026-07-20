package response_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-fries/fries/http/response/v4"
)

func Example() {
	recorder := httptest.NewRecorder()
	body := response.Success(
		"Scratch 11 is working properly.",
		map[string]any{
			"id":   11,
			"name": "Scratch 11",
		},
		response.WithCode(10000),
	)

	if err := response.Write(recorder, http.StatusOK, body); err != nil {
		panic(err)
	}

	fmt.Print(recorder.Body.String())
	// Output:
	// {"status":true,"code":10000,"message":"Scratch 11 is working properly.","data":{"id":11,"name":"Scratch 11"}}
}
