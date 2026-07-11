# Parallel

`parallel` provides small, context-aware helpers for concurrent batch work.
It propagates errors and cancellation, supports explicit concurrency limits,
and avoids background worker lifecycle management.

## Installation

```bash
go get github.com/go-fries/fries/parallel/v4
```

## Run tasks

```go
err := parallel.RunLimit(ctx, 4,
	func(ctx context.Context) error {
		return refreshCache(ctx)
	},
	func(ctx context.Context) error {
		return updateIndex(ctx)
	},
)
```

The first task error cancels the context passed to the remaining tasks. `Run`
provides the same behavior without a concurrency limit.

## Process collections

Use `ForEach` for concurrent side effects:

```go
err := parallel.ForEach(ctx, 8, userIDs, func(ctx context.Context, id int64) error {
	return notifyUser(ctx, id)
})
```

Use `Map` for type-safe transformations. Results preserve input order even when
callbacks finish in a different order:

```go
profiles, err := parallel.Map(ctx, 8, userIDs,
	func(ctx context.Context, id int64) (Profile, error) {
		return loadProfile(ctx, id)
	},
)
```

All functions wait for started work to return. Callbacks should observe the
provided context and stop promptly after cancellation.
