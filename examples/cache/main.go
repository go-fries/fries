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
	// 创建个 Redis 连接客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer rdb.Close()

	// create a redis store
	store := redisStore.New(rdb, redisStore.Prefix("example:cache"))

	// create a cache repository
	repository := cache.NewRepository(store)

	// set cache
	ok, err := repository.Set(ctx, "key", User{
		Name: "example",
		Age:  18,
	}, time.Second*10)
	if err != nil {
		log.Fatal(err)
	}
	_ = ok

	// get cache
	var user User
	err = repository.Get(ctx, "key", &user)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("user: %+v", user)

	// remember
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
