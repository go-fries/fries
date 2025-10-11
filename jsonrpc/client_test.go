package jsonrpc_test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-fries/fries/jsonrpc/v3"
)

type Result struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func TestClient(t *testing.T) {
	transport := jsonrpc.NewHTTPTransport(
		"https://localhost:8080",
		jsonrpc.WithHTTPTransportUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"),
	)
	client := jsonrpc.NewClient(transport)

	var result Result
	resp, err := client.Namespace("user/order").Call(t.Context(), &result, "salesDetail", 686812)
	spew.Dump(resp, err)

	t.Logf("%+v", result)
}
