package idempotency_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-fries/fries/idempotency/v4"
)

func ExampleExecutor_Do() {
	executor := idempotency.New(newExampleStore())
	calls := 0
	handler := func(context.Context) error {
		calls++
		return nil
	}

	_ = executor.Do(context.Background(), "orders:create:123", handler)
	_ = executor.Do(context.Background(), "orders:create:123", handler)

	fmt.Println(calls)

	// Output:
	// 1
}

type exampleStore struct {
	mu        sync.Mutex
	token     string
	completed bool
}

func newExampleStore() *exampleStore {
	return &exampleStore{}
}

func (s *exampleStore) Begin(
	_ context.Context,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.completed {
		return idempotency.BeginResult{Status: idempotency.BeginCompleted}, nil
	}
	if s.token != "" {
		return idempotency.BeginResult{Status: idempotency.BeginInProgress}, nil
	}
	s.token = request.Token
	return idempotency.BeginResult{Status: idempotency.BeginAcquired}, nil
}

func (s *exampleStore) Complete(
	_ context.Context,
	request idempotency.CompleteRequest,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Token != s.token {
		return idempotency.ErrClaimLost
	}
	s.completed = true
	s.token = ""
	return nil
}

func (s *exampleStore) Abort(
	_ context.Context,
	request idempotency.AbortRequest,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if request.Token != s.token {
		return idempotency.ErrClaimLost
	}
	s.token = ""
	return nil
}
