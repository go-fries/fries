package memory_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/idempotency/memory/v4"
	"github.com/go-fries/fries/idempotency/v4"
)

func Example() {
	executor := idempotency.New(memory.New())
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
