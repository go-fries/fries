# Hyperf Jet Retry Middleware

This module adapts the base `retry` component to Hyperf Jet middleware. Retry
execution, context cancellation, backoff, predicates, notifications, and final
error behavior are provided by `github.com/go-fries/fries/retry/v4`.

## Installation

```bash
go get github.com/go-fries/fries/hyperf/jet/middleware/retry/v4
```

## Basic usage

Use import aliases to distinguish the middleware from the base component:

```go
import (
	"time"

	"github.com/go-fries/fries/hyperf/jet/v4"
	baseretry "github.com/go-fries/fries/retry/v4"
	jetretry "github.com/go-fries/fries/hyperf/jet/middleware/retry/v4"
)

func configureRetry(client *jet.Client) {
	client.Use(jetretry.New(
		baseretry.WithMaxAttempts(3),
		baseretry.WithBackoff(
			baseretry.Exponential(100*time.Millisecond, time.Second),
		),
	))
}
```

The default configuration uses three total attempts and exponential backoff
starting at 100 milliseconds and capped at one second.

## Retryable errors

By default, the middleware retries:

- errors returned by the Jet timeout middleware;
- HTTP 408 Request Timeout;
- HTTP 429 Too Many Requests; and
- HTTP 5xx responses.

Other HTTP 4xx and unrelated errors are returned immediately. Override the
predicate with a base retry option:

```go
client.Use(jetretry.New(
	baseretry.WithRetryIf(func(err error) bool {
		return jetretry.DefaultRetryIf(err) ||
			errors.Is(err, ErrTemporaryBusinessFailure)
	}),
))
```

Only retry operations that are idempotent or otherwise safe to execute more
than once.

## Observe retries

Use the base component notification hook for logging or metrics:

```go
client.Use(jetretry.New(
	baseretry.WithNotify(func(ctx context.Context, event baseretry.Event) {
		logger.WarnContext(ctx, "Jet call will retry",
			"attempt", event.Attempt,
			"max_attempts", event.MaxAttempts,
			"delay", event.Delay,
			"error", event.Err,
		)
	}),
))
```

## Middleware order and timeouts

Place retry outside timeout when each attempt should receive an independent
timeout:

```go
client.Use(
	jetretry.New(),
	timeout.New(),
)
```

```text
retry -> timeout -> handler
```

Placing timeout outside retry gives the complete retry sequence one shared
timeout context:

```go
client.Use(
	timeout.New(),
	jetretry.New(),
)
```

```text
timeout -> retry -> handler
```

When the shared context expires, no further attempts are started.
