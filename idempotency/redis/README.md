# Idempotency Redis Store

`redis` provides a cross-process implementation of
`github.com/go-fries/fries/idempotency/v4.Store`.

## Installation

```bash
go get github.com/go-fries/fries/idempotency/redis/v4
```

## Usage

```go
client := redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
})

store := idempotencyredis.New(
	client,
	idempotencyredis.WithPrefix("billing:idempotency"),
)
executor := idempotency.New(store)

err := executor.Do(ctx, "orders:create:123", func(ctx context.Context) error {
	return createOrder(ctx)
})
```

The default prefix is `fries:idempotency:`. Configure an application-specific
prefix when projects share a Redis deployment.

Claims and completion records are stored as Redis hashes. Lua scripts perform
each state transition atomically, and Redis TTLs expire abandoned claims and
completed records. The Redis server must support scripting.

An execution may outlive its claim TTL. In that case another caller may start
the same operation, and the original caller receives
`idempotency.ErrClaimLost` when it attempts to complete. Choose an execution
TTL longer than the expected business operation and retain database-level
correctness controls for critical side effects.
