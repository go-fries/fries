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

func ExampleDoValue() {
	executor := idempotency.New(newExampleStore())
	calls := 0
	handler := func(context.Context) (exampleValue, error) {
		calls++
		return exampleValue{ID: 123}, nil
	}

	first, _ := idempotency.DoValue(
		context.Background(),
		executor,
		"orders:create:123",
		handler,
	)
	second, _ := idempotency.DoValue(
		context.Background(),
		executor,
		"orders:create:123",
		handler,
	)

	fmt.Println(first.Value.ID, first.Replayed)
	fmt.Println(second.Value.ID, second.Replayed)
	fmt.Println(calls)

	// Output:
	// 123 false
	// 123 true
	// 1
}

type exampleValue struct {
	ID int `json:"id"`
}

type exampleStore struct {
	mu        sync.Mutex
	token     string
	completed bool
	result    []byte
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
		return idempotency.BeginResult{
			Status: idempotency.BeginCompleted,
			Result: append([]byte(nil), s.result...),
		}, nil
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
	s.result = append([]byte(nil), request.Result...)
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
