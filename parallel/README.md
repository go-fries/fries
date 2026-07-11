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

## Keep partial results

Use `MapResults` when every item should be attempted even if some callbacks
fail. Each result corresponds to the input at the same index:

```go
results, batchErr := parallel.MapResults(ctx, 8, userIDs,
	func(ctx context.Context, id int64) (Profile, error) {
		return loadProfile(ctx, id)
	},
)

for index, result := range results {
	if result.Err != nil {
		log.Printf("load user %d: %v", userIDs[index], result.Err)
		continue
	}
	useProfile(result.Value)
}

if batchErr != nil {
	return batchErr
}
```

## Filter values

`Filter` evaluates a predicate concurrently and preserves input order:

```go
activeUsers, err := parallel.Filter(ctx, 8, users,
	func(ctx context.Context, user User) (bool, error) {
		return service.IsActive(ctx, user.ID)
	},
)
```

## Reuse fixed workers

Use `Pool` for intermittent work that should share a fixed concurrency limit
and a bounded queue:

```go
pool, err := parallel.NewPool(appContext, 8, parallel.WithQueueSize(32))
if err != nil {
	return err
}

future, err := pool.Submit(taskContext, func(ctx context.Context) error {
	return refreshCache(ctx, cacheKey)
})
if err != nil {
	return err
}

// Wait only when this request needs the task result.
if err := future.Wait(ctx); err != nil {
	return err
}
```

`Submit` returns after the task is accepted; execution may begin immediately or
after an earlier task finishes. `Execute` combines submission and waiting for
synchronous handlers.

The context passed to `Submit` is also passed to the task. If background work
must outlive a request, create an explicit task context, such as one derived
with `context.WithoutCancel`, instead of using the request context directly.
Task panics follow normal Go semantics and are not recovered by the pool.

Shut the pool down when its owner stops:

```go
shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := pool.Shutdown(shutdownContext); err != nil {
	return err
}
```

Batch helpers wait for started work to return. Pool submission returns after
acceptance and exposes completion through `Future`. Callbacks and tasks should
observe the provided context and stop promptly after cancellation.
