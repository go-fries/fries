package retry_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/retry/v4"
)

func ExampleDo() {
	attempts := 0
	err := retry.Do(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temporarily unavailable")
		}
		return nil
	}, retry.WithBackoff(retry.NoBackoff()))

	fmt.Println(attempts, err)
	// Output: 3 <nil>
}

func ExampleDoValue() {
	value, err := retry.DoValue(context.Background(), func(context.Context) (string, error) {
		return "ready", nil
	})

	fmt.Println(value, err)
	// Output: ready <nil>
}
