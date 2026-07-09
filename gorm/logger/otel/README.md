# GORM OpenTelemetry Logger

This package provides a GORM logger implementation backed by the [OpenTelemetry Logs API](https://opentelemetry.io/docs/specs/otel/logs/api/).

## Installation

```bash
go get github.com/go-fries/fries/gorm/logger/otel/v4
```

## Usage

```go
package main

import (
	"context"
	"time"

	"github.com/go-fries/fries/gorm/logger/otel/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDB(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		Logger: otel.New(
			otel.WithLoggerProvider(global.GetLoggerProvider()),
			otel.WithLogLevel(logger.Warn),
			otel.WithSlowThreshold(200*time.Millisecond),
			otel.WithParameterizedQueries(true),
			otel.WithAttributes(attribute.String("component", "gorm")),
			otel.WithLogAttributes(log.String("db.system", "mysql")),
			otel.WithLogAttributeFuncs(func(ctx context.Context) []log.KeyValue {
				return []log.KeyValue{
					log.String("tenant.id", tenantIDFromContext(ctx)),
				}
			}),
		),
	})
}

func tenantIDFromContext(ctx context.Context) string {
	return "tenant-1"
}
```

## Log Records

`Trace` emits SQL log records for errors, slow SQL, and info-level query
logging. SQL records include these attributes:

- `db.query.text`
- `gorm.rows_affected`
- `gorm.elapsed_ms`
- `gorm.event`

## Record Not Found

`logger.ErrRecordNotFound` is ignored by default because it is commonly an
expected query miss rather than an application failure.

Use `WithIgnoreRecordNotFoundError(false)` to report it as an error log record.

## Attributes

Use `WithLogAttributes` and `WithLogAttributeFuncs` to add fixed or
context-derived attributes to each emitted log record.

## SQL Parameters

Use `WithParameterizedQueries(true)` to keep GORM from expanding SQL parameter
values into the rendered query text.
