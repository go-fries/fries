# Kratos Multi Logger

This package provides a Kratos logger that dispatches each log call to multiple
`github.com/go-kratos/kratos/v2/log.Logger` implementations.

## Installation

```bash
go get github.com/go-fries/fries/kratos/log/multi/v3
```

## Usage

```go
package main

import (
	"os"

	"github.com/go-fries/fries/kratos/log/multi/v3"
	"github.com/go-kratos/kratos/v2/log"
)

func main() {
	logger := multi.New(
		log.NewStdLogger(os.Stdout),
		log.NewStdLogger(os.Stderr),
	)

	_ = logger.Log(log.LevelInfo, "key", "value")
}
```

All underlying loggers are called. If multiple loggers return errors, the
errors are combined with `errors.Join`.
