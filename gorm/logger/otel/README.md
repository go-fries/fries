# GORM OpenTelemetry Logger

This package provides a GORM logger implementation backed by the [OpenTelemetry Logs API](https://opentelemetry.io/docs/specs/otel/logs/api/).

## Installation

```bash
go get github.com/go-fries/fries/gorm/logger/otel/v3
```

## Usage

```go
package main

import (
	"time"

	gormotel "github.com/go-fries/fries/gorm/logger/otel/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openDB(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		Logger: gormotel.New(
			gormotel.WithLoggerProvider(global.GetLoggerProvider()),
			gormotel.WithLogLevel(gormlogger.Warn),
			gormotel.WithSlowThreshold(200*time.Millisecond),
			gormotel.WithParameterizedQueries(true),
			gormotel.WithAttributes(attribute.String("component", "gorm")),
		),
	})
}
```

`Trace` emits SQL log records for errors, slow SQL, and info-level query
logging. SQL records include `db.query.text`, `db.response.returned_rows`,
`gorm.rows_affected`, `gorm.elapsed_ms`, and `gorm.event` attributes.
Use `WithParameterizedQueries(true)` to keep GORM from expanding SQL parameter
values into the rendered query text.
