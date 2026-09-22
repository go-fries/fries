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
