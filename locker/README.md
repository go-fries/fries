# Locker

## Usage

```go
package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/go-fries/fries/cache/v4"
	redisStore "github.com/go-fries/fries/cache/redis/v4"
	redisLocker "github.com/go-fries/fries/locker/redis/v4"
)

func main() {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// ex1: 直接用
	locker := redisLocker.NewLocker(client, redisLocker.WithName("lock"), redisLocker.WithTTL(5*time.Minute))
	_ = locker.Try(context.Background(), func() {
		// do something
	})

	// ex2: 基于缓存用
	repository := cache.NewRepository(
		redisStore.New(client, redisStore.Prefix("cache")),
	)

	_ = repository.Lock("lock", 5*time.Minute).Try(context.Background(), func() {
		// do something
	})
}

```
