# Rate Limit Memory Store

`ratelimit/memory` stores GCRA state inside one Go process. It is suitable for
single-instance services, local development, and tests.

## Installation

```bash
go get github.com/go-fries/fries/ratelimit/v4
go get github.com/go-fries/fries/ratelimit/memory/v4
```

## Usage

```go
store := memory.New()
limiter, err := ratelimit.New(store, ratelimit.PerSecond(10))
if err != nil {
	return err
}

decision, err := limiter.Allow(ctx, "api:user:"+userID)
```

The Store is safe for concurrent use. Recovered keys are removed
opportunistically; it does not start a background goroutine and does not need
to be closed.

Each process has independent capacity. Use the Redis Store when multiple
application instances must enforce one shared limit.
