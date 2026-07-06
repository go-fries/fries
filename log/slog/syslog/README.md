# slog syslog handler

`log/slog/syslog` provides a `slog.Handler` that writes log records to syslog.

## Installation

```bash
go get github.com/go-fries/fries/log/slog/syslog/v4
```

## Usage

```go
package main

import (
	"log/slog"

	slogsyslog "github.com/go-fries/fries/log/slog/syslog/v4"
)

func main() {
	handler, err := slogsyslog.Dial("", "", slogsyslog.WithTag("api"))
	if err != nil {
		panic(err)
	}
	defer handler.Close()

	logger := slog.New(handler)
	logger.Info("service started", slog.String("service", "api"))
}
```

Use `NewHandler` with an injected writer when tests should not depend on a real
syslog service.
