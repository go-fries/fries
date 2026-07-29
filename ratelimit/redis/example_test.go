package redis_test

import (
	"github.com/go-fries/fries/ratelimit/redis/v4"
	"github.com/go-fries/fries/ratelimit/v4"
	goredis "github.com/redis/go-redis/v9"
)

func Example() {
	client := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = client.Close() }()

	store := redis.New(client, redis.WithPrefix("my-service:ratelimit"))
	_, _ = ratelimit.New(store, ratelimit.PerMinute(100))
}
