# Retry - Hyperf jet middleware

Retry middleware for Hyperf jet.

## Usage Example

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/go-fries/fries/hyperf/jet/v4"
	"github.com/go-fries/fries/hyperf/jet/middleware/retry/v4"
)

var customErr = errors.New("custom error")

func main() {
	client, err := jet.NewClient(
		jet.WithTransporter(nil),
		jet.WithService("Example/User/MoneyService"),
	)
	if err != nil {
		log.Fatal(err)
	}

	// base usage
	client.Use(retry.New())

	// with options
	client.Use(retry.New(
		// allow retry when custom error
		retry.Allow(func(err error) bool {
			return errors.Is(err, customErr)
		}),

		// or: allow retry with OrAllowFuncs
		retry.Allow(retry.OrAllowFuncs(
			retry.DefaultAllow,
			func(err error) bool {
				return errors.Is(err, customErr)
			}),
		// ... more allow
		),

		// retry 3 times
		retry.Attempts(3),

		// retry with backoff
		retry.Backoff(retry.LinearBackoff(100*time.Second)),
		// or: retry with NoBackoff
		retry.Backoff(retry.NoBackoff()),
		// or: retry with ExponentialBackoff
		retry.Backoff(retry.ExponentialBackoff(100*time.Second)),
		// or: retry with ConstantBackoff
		retry.Backoff(retry.ConstantBackoff(100*time.Second)),
	))

	// call service
	client.Invoke(context.Background(), "service", []any{"..."}, nil)
}
```
