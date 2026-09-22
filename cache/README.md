# Cache Component

A flexible caching library for Go applications, providing a unified API for various storage backends (like Redis) with support for serialization, atomic counters, and locking primitives.

## Installation

```bash
go get github.com/go-fries/fries/cache/v4
```

## Features

*   **Unified Interface:** Consistent API for different cache stores.
*   **Redis Support:** Built-in support for Redis via `go-redis`.
*   **Automatic Serialization:** seamless handling of complex Go types using JSON (or other codecs).
*   **Cache-aside `Remember`:** fetches cached values, or calls a callback and stores the result on cache miss.

## Recommended usage

Create the backend and Repository once, then pass the Repository to application
services. Use these entry points for common operations:

| Intent | Entry point |
| --- | --- |
| Read a typed value | `cache.Get[T](ctx, repository, key)` |
| Load and cache a value on a miss | `cache.Remember(ctx, repository, key, ttl, loader)` |
| Store a value | `repository.Set(ctx, key, value, ttl)` |
| Remove a value | `repository.Delete(ctx, key)` |
| Check existence | `repository.Has(ctx, key)` |
| Store a value only if absent | `repository.Add(ctx, key, value, ttl)` (atomicity depends on the backend) |

`Set` and `Delete` delegate to the Store's `Put` and `Forget`. Both pairs remain
supported. Use `repository.Get(ctx, key, &value)` when decoding into an existing
destination. A `Get` miss returns `ErrNotFound`; `Remember` invokes its loader
only on a miss, and concurrent misses may invoke that loader more than once.

The current write and delete methods return `(bool, error)`. Inspect the error
and use the status when it matters to the operation; for example, Redis deletion
returns `false, nil` when the key was already absent. `Add` is a conditional
write whose atomicity depends on the backend: the Repository fallback checks
existence before writing.

Configure Redis with its existing `Prefix` and `Codec` options. These names
remain supported even though new component options generally use `WithXxx`.
The example below shows setup, writes, reads, cache-aside loading and locking.

## Operation results

Check the error before interpreting a boolean result. Repository aliases return
the backend's result and error unchanged.

| Operation | `true, nil` | `false, nil` |
| --- | --- | --- |
| `Put` / `Set`, `Forever` | Backend reports a successful write. | Negative write status; success was not confirmed. |
| `Forget` / `Delete` | Backend reports deletion. | Key was already absent. |
| `Has` | Key exists. | Key does not exist. |
| `Add` | Backend reports insertion. | Key exists, or the fallback's `Put` reports a negative status. |
| `Flush` | Backend reports a successful clearing pass. | Backend reports a negative clearing status. |

A non-nil error means the operation could not be completed or confirmed. A write
may already have reached the backend before an error was observed, so an error
does not prove that the key was unchanged. These helpers preserve backend TTL
and encoding rules.

Redis reports successful `SET` operations as `true, nil`, and deleting an absent
key as `false, nil`. Its prefix-based `Flush` scans and removes matching keys;
success does not mean that the pass was atomic or that concurrent writers could
not add keys. `NullStore` is a no-op implementation: it reports existence and
successful writes, deletions and clearing without storing data.

## Conditional writes

`Repository.Add` calls the backend's `Add` when it implements `Addable`.
Otherwise, it checks existence before calling `Put`. That fallback is not
atomic: another writer can insert a value between the check and the write,
and the fallback can overwrite it.

The Redis adapter implements `Add` using `SET NX`, so its absence check and
write are atomic. When the key already exists, it returns `false, nil` and
leaves the value and expiration unchanged. For other backends, consult their
`Add` contract; implementing `Addable` alone does not guarantee atomicity.

## Usage

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	redisStore "github.com/go-fries/fries/cache/redis/v4"
	"github.com/go-fries/fries/cache/v4"
	"github.com/go-fries/fries/locker/v4"
)

var ctx = context.Background()

type User struct {
	Name string
	Age  int
}

func main() {
	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// Create a redis store
	store := redisStore.New(rdb, redisStore.Prefix("example:cache"))

	// Create a cache repository
	repository := cache.NewRepository(store)

	// Set cache
	ok, err := repository.Set(ctx, "key", User{
		Name: "example",
		Age:  18,
	}, time.Second*10)
	if err != nil {
		log.Fatal(err)
	}
	_ = ok

	// Get cache
	var user User
	err = repository.Get(ctx, "key", &user)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("user: %+v", user)

	// Remember: Get from cache, or execute function to get value and cache it
	user2, err := cache.Remember(ctx, repository, "key2", time.Second*10, func() (User, error) {
		return User{
			Name: "example2",
			Age:  20,
		}, nil
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("user2: %+v", user2)

	// Try to run work while holding a cache-prefixed Redis lock.
	err = locker.Try(ctx, repository.Lock("users:refresh", 30*time.Second), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Redis adapter tests

Run commands from the repository root:

```sh
# In-process tests; no Redis server is needed.
make test/cache/redis ARGS='-short -race -count=1'

# Full suite against a running Redis server.
REDIS_ADDR=localhost:6379 make test/cache/redis ARGS='-race -count=1'
```

`REDIS_ADDR` defaults to `localhost:6379`. Full-suite runs fail if Redis is unavailable. Each integration test uses a random key prefix, deletes only its declared cache and lock keys, and closes its client. Expiration checks observe the real Redis server with a bounded wait.

Unprefixed flush and cluster dispatch are tested with in-process command hooks. These tests run in short mode and verify SCAN/DEL requests without sending them to a server.
