# GORM Multi Logger

This package provides a GORM logger that dispatches each log call to multiple
`gorm.io/gorm/logger.Interface` implementations.

## Installation

```bash
go get github.com/go-fries/fries/gorm/logger/multi/v4
```

## Usage

```go
package main

import (
	"github.com/go-fries/fries/gorm/logger/multi/v4"
	"github.com/go-fries/fries/gorm/logger/otel/v4"
	"go.opentelemetry.io/otel/log/global"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDB(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		Logger: multi.New(
			logger.Default,
			otel.New(otel.WithLoggerProvider(global.GetLoggerProvider())),
		),
	})
}
```

`Trace` caches the result of GORM's SQL rendering callback before dispatching to
the underlying loggers, so the query text is rendered at most once per trace log
call.

If an underlying logger implements `gorm.ParamsFilter`, `ParamsFilter` is applied
in order. This allows filters such as parameter redaction and parameterized SQL
rendering to be composed.
