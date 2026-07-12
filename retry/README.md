# Retry

`retry` executes transiently failing operations with bounded, context-aware
retries. It supports typed results, retry filtering, reusable backoff
strategies, jitter, permanent errors, retry-after overrides, and retry
notifications.

## Installation

```bash
go get github.com/go-fries/fries/retry/v4
```

## Retry an operation

```go
err := retry.Do(ctx, func(ctx context.Context) error {
	return repository.Refresh(ctx)
})
```

The default configuration allows three total attempts and uses exponential
backoff starting at 100 milliseconds and capped at one second. The initial
execution counts as the first attempt.

Use options to make the retry behavior explicit for a call:

```go
err := retry.Do(ctx, func(ctx context.Context) error {
	return client.Send(ctx, request)
},
	retry.WithMaxAttempts(5),
	retry.WithBackoff(retry.Jitter(
		retry.Exponential(200*time.Millisecond, 5*time.Second),
		100*time.Millisecond,
	)),
	retry.WithRetryIf(func(err error) bool {
		return errors.Is(err, ErrUnavailable)
	}),
)
```

The caller's context controls the complete retry lifetime, including waits
between attempts:

```go
ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
defer cancel()

err := retry.Do(ctx, operation)
```

The operation itself must observe the supplied context while it is running.
If the context is canceled during an attempt and that attempt returns an error,
the context error takes precedence.

`Do` and `DoValue` panic when the operation is nil. As with other Go APIs that
accept `context.Context`, callers must not pass a nil context. Attempt counts
below one, nil backoff, and nil retry predicates leave the current configuration
unchanged. Backoff constructors and `After` panic on negative durations so
invalid static configuration fails immediately. A negative duration returned
dynamically by a custom backoff is treated as zero. The package does not recover
panics raised by an operation or notification callback.

## Return a value

Use `DoValue` for typed results:

```go
profile, err := retry.DoValue(ctx, func(ctx context.Context) (Profile, error) {
	return service.LoadProfile(ctx, userID)
})
```

When all attempts fail, `DoValue` returns the value and error produced by the
final execution.

## Stop retrying

Return `Permanent` for errors that another attempt cannot resolve:

```go
if errors.Is(err, ErrInvalidRequest) {
	return retry.Permanent(err)
}
```

`Do` and `DoValue` return the underlying error, so normal `errors.Is` and
`errors.As` checks continue to work. Errors wrapped around the `Permanent`
marker are preserved.

## Override the next delay

Use `After` when a service supplies an explicit retry interval, such as an HTTP
`Retry-After` response:

```go
return retry.After(delay, ErrRateLimited)
```

The override still respects the attempt limit, retry predicate, and context.
Errors wrapped around the `After` marker are preserved.

## Observe scheduled retries

```go
err := retry.Do(ctx, operation,
	retry.WithNotify(func(ctx context.Context, event retry.Event) {
		logger.WarnContext(ctx, "operation will retry",
			"attempt", event.Attempt,
			"max_attempts", event.MaxAttempts,
			"delay", event.Delay,
			"error", event.Err,
		)
	}),
)
```

Notifications run synchronously before the wait for the next attempt. A
notification callback should return quickly and must be safe for concurrent use
when shared by multiple callers.

`retry` handles in-process operation retries only. Durable message redelivery,
acknowledgement, and dead-letter behavior remain responsibilities of queue
implementations.
