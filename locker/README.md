# Locker

Locker provides a small set of contracts for named, expiring locks and a Redis
implementation with ownership-safe release and explicit renewal.

## Installation

```bash
go get github.com/go-fries/fries/locker/v4
go get github.com/go-fries/fries/locker/redis/v4
```

## Run work while holding a lock

Use `locker.Do` when the caller should wait for the lock. The Context controls
how long acquisition may wait.

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-fries/fries/locker/v4"
	lockerredis "github.com/go-fries/fries/locker/redis/v4"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	backend := lockerredis.New(client)
	lock := backend.Lock("orders:123", 30*time.Second)

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := locker.Do(waitCtx, lock, func(ctx context.Context) error {
		// Perform synchronous work while the lease is held.
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

Use `locker.Try` for a single acquisition attempt. It returns
`locker.ErrNotAcquired` when another lease owns the lock.

## Manage a lease explicitly

Acquire a lease directly when you need renewal, token transfer, a custom
release Context, or control over when release happens.

```go
func runWithLease(ctx context.Context, lock locker.Lock) error {
	lease, err := lock.Acquire(ctx)
	if err != nil {
		return err
	}

	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := lease.Release(releaseCtx); err != nil {
			log.Printf("release lock: %v", err)
		}
	}()

	if renewable, ok := lease.(locker.RenewableLease); ok {
		if err := renewable.Refresh(ctx, 30*time.Second); err != nil {
			return err
		}
	}

	return nil
}
```

`locker.Do` and `locker.Try` release synchronously with
`context.WithoutCancel(ctx)`. This preserves Context values while allowing a
release attempt after the business Context is canceled. They join handler and
release errors. Use an explicit lease when release must have its own deadline.

## Redis lease boundaries

A Redis lock is an expiring lease, not permanent ownership. Work that outlives
the TTL may overlap with a new holder. Explicit renewal reduces that risk but
does not eliminate process pauses, network partitions, or asynchronous Redis
failover behavior. Correctness-sensitive applications should also enforce
idempotency or constraints at the protected resource. The package does not
provide fencing tokens; rejecting stale workers requires a monotonic token that
the protected resource validates atomically.
