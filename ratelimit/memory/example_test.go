package memory_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-fries/fries/ratelimit/memory/v4"
	"github.com/go-fries/fries/ratelimit/v4"
)

func Example() {
	store := memory.New()
	limiter, _ := ratelimit.New(store, ratelimit.Limit{
		Rate:   1,
		Period: time.Minute,
		Burst:  2,
	})

	first, _ := limiter.Allow(context.Background(), "user:123")
	second, _ := limiter.Allow(context.Background(), "user:123")
	third, _ := limiter.Allow(context.Background(), "user:123")

	fmt.Println(first.Allowed, second.Allowed, third.Allowed)

	// Output:
	// true true false
}
