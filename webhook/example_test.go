package webhook_test

import (
	"fmt"

	"github.com/go-fries/fries/webhook/v4"
)

func Example() {
	secret, err := webhook.NewSecret(
		[]byte("01234567890123456789012345678901"),
	)
	if err != nil {
		panic(err)
	}

	signer, err := webhook.NewSigner(secret)
	if err != nil {
		panic(err)
	}
	verifier, err := webhook.NewVerifier(secret)
	if err != nil {
		panic(err)
	}

	payload := []byte(`{"type":"order.created"}`)
	headers, err := signer.Sign("msg_123", payload)
	if err != nil {
		panic(err)
	}

	metadata, err := verifier.Verify(headers, payload)
	fmt.Println(metadata.ID, err)

	// Output:
	// msg_123 <nil>
}
