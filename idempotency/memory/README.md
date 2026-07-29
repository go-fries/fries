# Idempotency Memory Store

`memory` provides an in-process implementation of
`github.com/go-fries/fries/idempotency/v4.Store`.

## Installation

```bash
go get github.com/go-fries/fries/idempotency/memory/v4
```

## Usage

```go
store := memory.New()
executor := idempotency.New(store)

err := executor.Do(ctx, "orders:create:123", func(ctx context.Context) error {
	return createOrder(ctx)
})
```

The store is concurrency-safe and expires records lazily without background
goroutines. Records are process-local and are lost when the process exits, so
this adapter is suitable for tests and single-process workloads. Use the Redis
adapter when multiple application instances must coordinate the same keys.
