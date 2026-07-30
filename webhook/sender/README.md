# Webhook Sender

`webhook/sender` performs one signed HTTP POST delivery to a configured
endpoint. It combines `webhook.Signer` with safe HTTP defaults while leaving
durable retries and delivery settlement to the application or queue.

## Installation

```bash
go get github.com/go-fries/fries/webhook/v4
go get github.com/go-fries/fries/webhook/sender/v4
```

## Usage

Bind one endpoint to its signing secret:

```go
secret, err := webhook.ParseSecret(os.Getenv("WEBHOOK_SECRET"))
if err != nil {
	return err
}
signer, err := webhook.NewSigner(secret)
if err != nil {
	return err
}
outbound, err := sender.New(
	"https://customer.example.com/webhooks",
	signer,
)
if err != nil {
	return err
}
```

Send one message:

```go
result, err := outbound.Send(ctx, sender.Message{
	ID:      event.ID,
	Payload: payload,
	Header: http.Header{
		"content-type": {"application/json"},
	},
})
if err != nil {
	return err
}
if !result.Successful() {
	return fmt.Errorf("webhook returned HTTP %d", result.StatusCode)
}
```

The message ID should remain stable across retries. Sender generates a current
timestamp and signature immediately before every HTTP attempt.

## HTTP behavior

Sender:

- performs exactly one HTTP POST per `Send` call;
- requires HTTPS by default;
- never follows redirects;
- uses a 30-second total request timeout by default;
- defaults Content-Type to `application/json`;
- overwrites all Standard Webhooks signature headers;
- drains at most 64 KiB of the response body and closes it;
- returns response status, headers, and a parsed `Retry-After` delay.

Configure a custom HTTP client and timeout when required:

```go
outbound, err := sender.New(
	endpoint,
	signer,
	sender.WithHTTPClient(client),
	sender.WithTimeout(20*time.Second),
)
```

The supplied client is copied. Sender disables redirects on the copy and does
not modify the original client. A positive client timeout is preserved unless
`WithTimeout` overrides it; a client without a timeout receives the 30-second
default.

HTTP can be enabled explicitly for trusted development or private-network
environments:

```go
outbound, err := sender.New(
	"http://localhost:8080/webhook",
	signer,
	sender.WithInsecureHTTP(),
)
```

## Result and retries

Transport failures, Context cancellation, and request construction failures
are returned as errors. An HTTP response is returned as a `Result`, including
4xx and 5xx responses:

```go
result, err := outbound.Send(ctx, message)
if err != nil {
	return err
}

switch {
case result.Successful():
	return nil
case result.StatusCode == http.StatusGone:
	return queue.DeadLetter("webhook endpoint is gone")
case result.RetryAfter > 0:
	return queue.RetryAfter(result.RetryAfter)
case result.StatusCode == http.StatusTooManyRequests:
	return errors.New("webhook rate limited")
case result.StatusCode >= 500:
	return errors.New("webhook server error")
default:
	return queue.DeadLetter("webhook rejected")
}
```

`webhook/sender` does not depend on `queue`; the example shows how an
application may map delivery results to its own queue policy.

Do not assume a transport error means the endpoint did not process the
request. A connection may fail after the receiver committed the operation.
Retries can therefore deliver the same message more than once, and receivers
should use `webhook-id` as an idempotency key.

## Endpoint security

Sender rejects malformed endpoints, URL credentials, fragments, unsupported
schemes, and HTTP unless it is explicitly enabled. It also supports
application-specific validation:

```go
outbound, err := sender.New(
	endpoint,
	signer,
	sender.WithEndpointValidator(func(
		ctx context.Context,
		endpoint *url.URL,
	) error {
		if !allowedHosts[endpoint.Hostname()] {
			return ErrEndpointNotAllowed
		}
		return nil
	}),
)
```

URL validation alone is not complete SSRF protection. When customers control
endpoints, use an outbound proxy or network policy that prevents access to
loopback addresses, private networks, link-local addresses, and cloud metadata
services. DNS must be checked at connection time to protect against rebinding.

## Boundaries

Sender does not:

- persist messages or delivery attempts;
- automatically retry failures;
- decide whether a status code is retryable;
- manage endpoint lifecycle or disable failed endpoints;
- retain or expose response bodies;
- provide complete SSRF protection;
- guarantee exactly-once delivery.
