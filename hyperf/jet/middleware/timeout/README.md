# Timeout - Jet Middleware

Timeout middleware for Jet.

## Usage Example

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/go-fries/fries/hyperf/jet/v4"
	"github.com/go-fries/fries/hyperf/jet/middleware/timeout/v4"
)

func main() {
	client, err := jet.NewClient(
		jet.WithTransporter(nil),
		// ...
	)
	if err != nil {
		log.Fatal(err)
	}

	// base usage
	client.Use(timeout.New()) // default 5s

	// custom timeout
	client.Use(timeout.New(
		timeout.Timeout(10 * time.Second),
	))

	// call service
	client.Invoke(context.Background(), "method", []any{"..."}, nil)
}
```
