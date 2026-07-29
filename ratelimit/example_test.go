package ratelimit_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-fries/fries/ratelimit/v4"
)

func ExampleLimiter_Allow() {
	limiter, _ := ratelimit.New(&exampleStore{}, ratelimit.PerMinute(1))

	first, _ := limiter.Allow(context.Background(), "api:user:123")
	second, _ := limiter.Allow(context.Background(), "api:user:123")

	fmt.Println(first.Allowed)
	fmt.Println(second.Allowed)

	// Output:
	// true
	// false
}

type exampleStore struct {
	mu    sync.Mutex
	taken map[string]bool
}

func (s *exampleStore) Take(
	_ context.Context,
	request ratelimit.TakeRequest,
) (ratelimit.Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taken == nil {
		s.taken = make(map[string]bool)
	}
	if s.taken[request.Key] {
		return ratelimit.Decision{Limit: request.Limit}, nil
	}
	s.taken[request.Key] = true
	return ratelimit.Decision{
		Limit:   request.Limit,
		Allowed: true,
	}, nil
}

func (s *exampleStore) Reset(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.taken, key)
	return nil
}
