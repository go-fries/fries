# Redis Locker

Redis Locker implements the `locker` contracts with Redis. It provides named
expiring locks, ownership-safe release, cancellable acquisition waits, explicit
renewal, and ownership-token transfer.

## Installation

```bash
go get github.com/go-fries/fries/locker/redis/v4
```

## Usage

Create one reusable backend, then create a `Lock` for each resource name and
lease TTL.

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

	backend := lockerredis.New(
		client,
		lockerredis.WithPrefix("billing:locker"),
	)
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

`locker.Do` waits until acquisition succeeds or the Context ends. Use
`locker.Try` when only one acquisition attempt should be made. A competing lock
causes `locker.Try` or `TryAcquire` to return `locker.ErrNotAcquired`.
`lockerredis.New` panics when given a nil Redis client.

## Configure the key prefix

Redis keys use the `locker:` prefix by default. Applications sharing a Redis
deployment should set a project-specific prefix to prevent unrelated projects
from competing for the same lock name:

```go
backend := lockerredis.New(
	client,
	lockerredis.WithPrefix("billing:locker"),
)
```

The resulting Redis key for `orders:123` is
`billing:locker:orders:123`. Trailing colons are normalized. An empty prefix is
ignored and retains the default. The prefix is part of the lock identity, so a
process restoring a transferred lease must use the same prefix and lock name.

## Configure acquisition waits

`Acquire` waits for a competing lease by choosing a random interval between 50
and 100 milliseconds. Configure a different range when the workload needs it:

```go
backend := lockerredis.New(
	client,
	lockerredis.WithWaitInterval(20*time.Millisecond, 80*time.Millisecond),
)
```

Equal minimum and maximum values configure a fixed interval. Invalid intervals
are ignored. Waiting uses the acquisition Context, so cancellation and
deadlines interrupt it without waiting for the current interval to finish.

## Renew a lease

Acquire explicitly when work may outlive the original TTL. Redis leases
implement `locker.RenewableLease`:

```go
package example

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/locker/v4"
)

func runWithRenewal(ctx context.Context, lock locker.Lock) (err error) {
	lease, err := lock.Acquire(ctx)
	if err != nil {
		return err
	}

	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		err = errors.Join(err, lease.Release(releaseCtx))
	}()

	renewable := lease.(locker.RenewableLease)
	return renewable.Refresh(ctx, 30*time.Second)
}
```

`Refresh` resets expiration to the supplied TTL from the time Redis processes
the request. It does not add the TTL to the remaining time and does not start a
background renewal goroutine. If the token no longer owns the lock, it returns
`locker.ErrLeaseLost`.

## Transfer ownership

Redis leases expose their own token through `locker.TransferableLease`. A
different process can restore a lease for the same lock name through
`locker.RestorableLock`:

```go
package example

import (
	"time"

	"github.com/go-fries/fries/locker/v4"
)

func transferToken(lease locker.Lease) string {
	return lease.(locker.TransferableLease).Token()
}

func restoreLease(backend locker.Locker, token string) (locker.Lease, error) {
	// The receiving process must reconstruct the same named lock.
	lock := backend.Lock("orders:123", 30*time.Second)
	return lock.(locker.RestorableLock).Restore(token)
}
```

Treat the token as a secret capability. `Restore` does not contact Redis or
pre-check ownership; `Release` and `Refresh` perform the actual atomic token
check. An empty token returns `locker.ErrInvalidToken`.

## Error semantics

- Lock contention during a single attempt returns `locker.ErrNotAcquired`.
- Acquisition cancellation and timeout return `context.Canceled` and
  `context.DeadlineExceeded`.
- Releasing or refreshing an expired or replaced lease returns
  `locker.ErrLeaseLost`.
- Empty names and non-positive TTLs return `locker.ErrInvalidName` and
  `locker.ErrInvalidTTL` before Redis is accessed.
- Redis connection, protocol, and script failures preserve the underlying
  error for `errors.Is` and `errors.As`.

## Correctness boundaries

A Redis lock is an expiring lease. If work exceeds the TTL, another process may
acquire the same name while the original work is still running. Renewal cannot
eliminate process pauses, network partitions, or the effects of asynchronous
Redis failover.

For correctness-sensitive operations, also enforce idempotency or constraints
at the protected resource. This implementation does not provide fencing tokens
or multi-node Redlock arbitration.
