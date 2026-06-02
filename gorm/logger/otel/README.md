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
	"context"
	"time"

	"github.com/go-fries/fries/gorm/logger/otel/v3"
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
			otel.WithTraceContext(),
			otel.WithLogAttributeFuncs(func(ctx context.Context) []log.KeyValue {
				return []log.KeyValue{
					log.String("tenant.id", tenantIDFromContext(ctx)),
				}
			}),
		),
	})
}

func tenantIDFromContext(context.Context) string {
	return "tenant-1"
}
```

`Trace` emits SQL log records for errors, slow SQL, and info-level query
logging. SQL records include `db.query.text`, `gorm.rows_affected`,
`gorm.elapsed_ms`, and `gorm.event` attributes.
`logger.ErrRecordNotFound` is ignored by default because it is commonly an
expected query miss rather than an application failure. Use
`WithIgnoreRecordNotFoundError(false)` to report it as an error log record.
Use `WithLogAttributes` and `WithLogAttributeFuncs` to add fixed or
context-derived attributes to each emitted log record.
Use `WithTraceContext` to add `trace.id` and `span.id` from the current span
context when they are available.
Use `WithParameterizedQueries(true)` to keep GORM from expanding SQL parameter
values into the rendered query text.
