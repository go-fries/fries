# slog multi handler

`log/slog/multi` provides a `slog.Handler` that fans out each enabled log record
to multiple child handlers.

## Installation

```bash
go get github.com/go-fries/fries/log/slog/multi/v4
```

## Usage

```go
package main

import (
	"log/slog"
	"os"

	slogmulti "github.com/go-fries/fries/log/slog/multi/v4"
)

func main() {
	handler := slogmulti.NewHandler(
		slog.NewTextHandler(os.Stdout, nil),
		slog.NewJSONHandler(os.Stderr, nil),
	)

	logger := slog.New(handler)
	logger.Info("service started", slog.String("service", "api"))
}
```

Nil handlers are ignored. A record is handled by each child handler that is
enabled for the record level. When multiple child handlers return errors, the
errors are combined with `errors.Join`.
