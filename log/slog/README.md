# slog provider

`log/slog` provides a lifecycle provider that sets the process-wide
`slog.Default()` logger during application bootstrap.

## Installation

```bash
go get github.com/go-fries/fries/log/slog/v4
```

## Usage

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-fries/fries/lifecycle/v4"
	slogprovider "github.com/go-fries/fries/log/slog/v4"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	runner := lifecycle.New(
		lifecycle.WithProviders(slogprovider.NewProvider(logger)),
	)

	if err := runner.Run(context.Background(), func(ctx context.Context) error {
		slog.InfoContext(ctx, "service started")
		return nil
	}); err != nil {
		panic(err)
	}
}
```

`Bootstrap` replaces the process-wide default logger. `Shutdown` does not
restore the previous logger.
