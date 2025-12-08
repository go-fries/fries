# Cache Component

A flexible caching library for Go applications, providing a unified API for various storage backends (like Redis) with support for serialization and atomic operations.

## Installation

```bash
go get github.com/go-fries/fries/cache/v3
```

## Features

*   **Unified Interface:** Consistent API for different cache stores.
*   **Redis Support:** Built-in support for Redis via `go-redis`.
*   **Automatic Serialization:** seamless handling of complex Go types using JSON (or other codecs).
*   **Atomic `Remember`:** preventing cache stampedes by atomically fetching and setting values if missing.

## Usage

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	redisStore "github.com/go-fries/fries/cache/redis/v3"
	"github.com/go-fries/fries/cache/v3"
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
}
```