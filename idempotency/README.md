# Idempotency

`idempotency` coordinates repeated business operations through a stable key.
Only the caller that acquires the key runs the handler. Concurrent callers
receive `ErrInProgress`, and calls made after successful completion return
without running the handler again.

## Installation

Choose a Store for the deployment model:

```bash
go get github.com/go-fries/fries/idempotency/v4
go get github.com/go-fries/fries/idempotency/memory/v4
```

Use `idempotency/redis` instead of `idempotency/memory` when multiple
application instances must coordinate the same keys.

## Usage

Create one Executor and reuse it across requests:

```go
store := memory.New()
executor := idempotency.New(
	store,
	idempotency.WithExecutionTTL(time.Minute),
	idempotency.WithResultTTL(24*time.Hour),
)

err := executor.Do(ctx, "orders:create:123", func(ctx context.Context) error {
	return createOrder(ctx)
})
```

The first call claims the key and runs the handler. A repeated call after
completion returns `nil` without running the handler. If another caller still
owns the claim, `Do` returns `idempotency.ErrInProgress`.

Use `DoValue` when repeated callers need the value produced by the first
successful execution:

```go
result, err := idempotency.DoValue(
	ctx,
	executor,
	"orders:create:"+idempotencyKey,
	func(ctx context.Context) (Order, error) {
		return createOrder(ctx, request)
	},
)
if err != nil {
	return err
}

order := result.Value
replayed := result.Replayed
```

The first call encodes and stores the returned value. A completed call decodes
the stored value, sets `Replayed` to `true`, and does not run the handler.
Values use `codec/json` by default. `WithCodec` accepts the shared
`codec.Codec` interface:

```go
executor := idempotency.New(
	store,
	idempotency.WithCodec(msgpack.Codec{}),
)
```

Use a stable result type and Codec for the lifetime of a key. Changing either
can make an existing completed result impossible to decode. Do not mix `Do`
and `DoValue` for the same key.

Use a fingerprint when the same key must never be accepted for different
input:

```go
err := executor.Do(
	ctx,
	"payments:confirm:"+idempotencyKey,
	func(ctx context.Context) error {
		return confirmPayment(ctx, request)
	},
	idempotency.WithFingerprint(requestFingerprint),
)
```

Reusing that key with a different non-empty fingerprint returns
`idempotency.ErrKeyConflict`.

## Common integrations

Queue handlers can use a stable business identifier rather than a delivery
attempt number:

```go
return executor.Do(ctx, "queue:charge:"+orderID, func(ctx context.Context) error {
	return chargeOrder(ctx, orderID)
})
```

HTTP handlers can read an `Idempotency-Key`, validate it, and choose the
response for coordination errors:

```go
err := executor.Do(ctx, "http:orders:"+idempotencyKey, create)
switch {
case errors.Is(err, idempotency.ErrInProgress):
	// Return a conflict or retryable response.
case errors.Is(err, idempotency.ErrKeyConflict):
	// Reject reuse of the key for different input.
}
```

Webhook handlers can use a provider event ID:

```go
err := executor.Do(ctx, "webhook:stripe:"+event.ID, applyEvent)
```

HTTP responses, queue settlement, retry policy, and key generation remain the
application's responsibility.

## Execution semantics

- Execution claims and completed records have separate TTLs.
- A handler error aborts its claim so a later call can retry.
- If a `DoValue` handler succeeds but encoding fails, the claim remains until
  its execution TTL expires rather than immediately allowing duplicate side
  effects.
- Handler panics are not recovered. The claim remains until its execution TTL
  expires.
- Completion and abort use a short context detached from request cancellation
  while preserving context values.
- An expired claim may be acquired while its original handler is still
  running. The original caller then receives `ErrClaimLost` when it attempts
  to complete.

`idempotency` does not provide exactly-once execution or a distributed
transaction between the Store and business side effects. A handler may
succeed while saving the completion record fails. Correctness-sensitive
operations should still use database unique constraints, transactions, or a
downstream idempotency mechanism.
