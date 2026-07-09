# Logger - Hyperf jet middleware

Logger middleware for Hyperf jet.

## Usage Example

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-fries/fries/hyperf/jet/v4"
	"github.com/go-fries/fries/hyperf/jet/middleware/logger/v4"
)

func main() {
	client, err := jet.NewClient(
		jet.WithTransporter(nil),
		jet.WithService("Example/User/MoneyService"),
	)
	if err != nil {
		panic(err)
	}

	// base usage
	client.Use(logger.New()) // use slog.Default()

	// with options
	client.Use(logger.New(
		logger.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
	))

	// call service
	client.Invoke(context.Background(), "service", []any{"..."}, nil)
}
```
