# OpenTelemetry Logger for Kratos

This package provides a Kratos `log.Logger` implementation backed by the OpenTelemetry Logs API.

## Installation

```bash
go get github.com/go-fries/fries/kratos/log/otel/v4
```

## Usage

```go
package main

import (
	kratoslog "github.com/go-kratos/kratos/v2/log"
	otelkratos "github.com/go-fries/fries/kratos/log/otel/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log/global"
)

func main() {
	logger := otelkratos.NewLogger(
		otelkratos.WithLoggerProvider(global.GetLoggerProvider()),
		otelkratos.WithSchemaURL("https://opentelemetry.io/schemas/1.37.0"),
		otelkratos.WithAttributes(attribute.String("service.name", "example")),
	)
	logger = kratoslog.With(logger,
		"trace_id", otelkratos.TraceID(),
		"span_id", otelkratos.SpanID(),
	)

	helper := kratoslog.NewHelper(logger)
	helper.Infow("msg", "server started")
}
```

## Options

- `WithLoggerProvider` sets the OpenTelemetry logger provider.
- `WithVersion` sets the instrumentation scope version. The default is `otel.Version()`.
- `WithSchemaURL` sets the OpenTelemetry schema URL.
- `WithAttributes` adds instrumentation scope attributes.

`TraceID` and `SpanID` return Kratos log valuers that read the active
OpenTelemetry span from the log context.

The instrumentation scope name is fixed to the module import path:

```text
github.com/go-fries/fries/kratos/log/otel/v4
```
