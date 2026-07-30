# Webhook

`webhook` signs and verifies HTTP webhook messages using the
[Standard Webhooks](https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md)
HMAC-SHA256 scheme. It protects the message ID, timestamp, and exact payload
bytes without binding applications to a particular HTTP framework.

## Installation

```bash
go get github.com/go-fries/fries/webhook/v4
```

## Secret

Create a Secret from raw key bytes:

```go
// Example only. Load a random key from a secret manager in production.
secret, err := webhook.NewSecret(
	[]byte("0123456789abcdef0123456789abcdef"),
)
```

The key must contain between 24 and 64 bytes. `NewSecret` copies the key and
does not retain the caller's slice.

Standard Webhooks secrets can also be loaded from configuration:

```go
secret, err := webhook.ParseSecret(os.Getenv("WEBHOOK_SECRET"))
```

The encoded value uses `whsec_<base64>` format. Generate and store signing
keys with a cryptographically secure secret-management process. Use a
different key for each receiving endpoint.

## Sign a message

Create and reuse one Signer:

```go
signer, err := webhook.NewSigner(secret)
if err != nil {
	return err
}

payload := []byte(`{"type":"order.created","data":{"id":"123"}}`)

headers, err := signer.Sign("msg_123", payload)
if err != nil {
	return err
}

request, err := http.NewRequestWithContext(
	ctx,
	http.MethodPost,
	endpoint,
	bytes.NewReader(payload),
)
if err != nil {
	return err
}
request.Header = headers.Clone()
request.Header.Set("Content-Type", "application/json")
```

`Sign` returns `Webhook-Id`, `Webhook-Timestamp`, and `Webhook-Signature`.
The message ID must contain only visible ASCII characters other than `.`.

Sign the same payload bytes that will be sent. Formatting or marshaling the
payload again after signing can invalidate the signature.

## Verify a request

Create and reuse one Verifier:

```go
verifier, err := webhook.NewVerifier(secret)
if err != nil {
	return err
}
```

Read a bounded raw request body before decoding JSON:

```go
body, err := io.ReadAll(http.MaxBytesReader(w, request.Body, 1<<20))
if err != nil {
	http.Error(w, "invalid body", http.StatusBadRequest)
	return
}

metadata, err := verifier.Verify(request.Header, body)
if err != nil {
	http.Error(w, "invalid webhook", http.StatusUnauthorized)
	return
}

var event struct {
	Type string `json:"type"`
}
if err := json.Unmarshal(body, &event); err != nil {
	http.Error(w, "invalid payload", http.StatusBadRequest)
	return
}
```

Use the exact bytes read from the request. Do not decode and re-encode the
payload before verification.

The default timestamp tolerance is five minutes. Configure a different
positive tolerance when required:

```go
verifier, err := webhook.NewVerifier(
	secret,
	webhook.WithTolerance(10*time.Minute),
)
```

Requests older than the tolerance or too far in the future are rejected.
Timestamp validation limits old replays, but it does not prevent a valid
request from being delivered more than once within that window.

## Prevent duplicate processing

Use the authenticated message ID as part of a business idempotency key:

```go
metadata, err := verifier.Verify(request.Header, body)
if err != nil {
	return err
}

err = executor.Do(
	request.Context(),
	"webhook:orders:"+metadata.ID,
	func(ctx context.Context) error {
		return handleOrderEvent(ctx, body)
	},
)
```

Choose an idempotency TTL based on the business redelivery period. It does not
need to equal the webhook timestamp tolerance.

## Rotate secrets

During a rotation, producers can sign with both the current and previous
secrets:

```go
signer, err := webhook.NewSigner(
	current,
	webhook.WithAdditionalSigningSecrets(previous),
)
```

Consumers can accept both:

```go
verifier, err := webhook.NewVerifier(
	current,
	webhook.WithAdditionalVerificationSecrets(previous),
)
```

Remove the previous secret after deployments and outstanding deliveries have
passed the agreed transition period.

## Reliable delivery

This package does not send HTTP requests, persist delivery records, or retry
failed requests. For durable delivery, put the stable message ID and payload
in a queue and sign them immediately before each HTTP attempt:

```go
func deliver(
	ctx context.Context,
	endpoint string,
	messageID string,
	payload []byte,
) error {
	headers, err := signer.Sign(messageID, payload)
	if err != nil {
		return err
	}
	return send(ctx, endpoint, headers, payload)
}
```

Signing immediately before the request ensures a delayed or retried delivery
receives a current timestamp.

## Compatibility and boundaries

The package implements the Standard Webhooks symmetric `v1` signature format:

```text
Webhook-Id: msg_123
Webhook-Timestamp: 1700000000
Webhook-Signature: v1,<base64-hmac-sha256>
```

GitHub, Stripe, Slack, and other providers use their own header and signature
formats. Their deliveries cannot be verified directly with this package.

`webhook` does not:

- decode or define the payload schema;
- guarantee exactly-once processing;
- dispatch application events;
- send, persist, or retry deliveries;
- log payloads, secrets, or signatures;
- provide framework middleware or provider-specific adapters.

Signer and Verifier are safe for concurrent use and start no background
goroutines.
