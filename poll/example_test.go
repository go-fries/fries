package poll_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-fries/fries/poll/v4"
)

func ExampleUntil() {
	checks := 0
	err := poll.Until(context.Background(), time.Millisecond, func(context.Context) (bool, error) {
		checks++
		return checks == 3, nil
	})

	fmt.Println(checks, err)
	// Output: 3 <nil>
}

func ExampleUntilValue() {
	job, err := poll.UntilValue(
		context.Background(),
		time.Millisecond,
		func(context.Context) (string, bool, error) {
			return "completed", true, nil
		},
	)

	fmt.Println(job, err)
	// Output: completed <nil>
}
