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

	"github.com/go-fries/fries/foundation/v4"
	slogprovider "github.com/go-fries/fries/log/slog/v4"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	providers := foundation.NewChain(slogprovider.NewProvider(logger))

	ctx, err := providers.Bootstrap(context.Background())
	if err != nil {
		panic(err)
	}

	slog.InfoContext(ctx, "service started")

	if _, err := providers.Terminate(ctx); err != nil {
		panic(err)
	}
}
```

`Bootstrap` replaces the process-wide default logger. `Terminate` does not
restore the previous logger.
