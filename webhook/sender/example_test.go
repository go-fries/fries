package sender_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-fries/fries/webhook/sender/v4"
	"github.com/go-fries/fries/webhook/v4"
)

func ExampleSender_Send() {
	secret, err := webhook.NewSecret(
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		panic(err)
	}
	signer, err := webhook.NewSigner(secret)
	if err != nil {
		panic(err)
	}
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}
	value, err := sender.New(
		"https://customer.example.com/webhooks",
		signer,
		sender.WithHTTPClient(client),
	)
	if err != nil {
		panic(err)
	}

	result, err := value.Send(context.Background(), sender.Message{
		ID:      "msg_123",
		Payload: []byte(`{"type":"order.created"}`),
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = result.Body.Close()
	}()
	fmt.Println(result.StatusCode, err)

	// Output:
	// 204 <nil>
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return f(request)
}
