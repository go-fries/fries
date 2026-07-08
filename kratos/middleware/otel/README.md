# Kratos OpenTelemetry Middleware

The tracing middleware traces Kratos client and server requests with [OpenTelemetry](https://opentelemetry.io/).

The package is forked from [tracing](https://github.com/go-kratos/kratos/tree/8b8dc4b0f8bebb76939780f59734c20c265669c5/middleware/tracing) and optimized on this basis. Thanks to the original author for his contribution.

## Installation

```bash
go get github.com/go-fries/fries/kratos/middleware/otel/v4
```

## Usage

```go
package main

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport/http"

	kratosotel "github.com/go-fries/fries/kratos/middleware/otel/v4"
)

func main() {
	tracerProvider := otel.GetTracerProvider()

	app := kratos.New(
		kratos.Name("tracing"),
		kratos.Server(
			http.NewServer(
				http.Address(":8001"),
				http.Middleware(kratosotel.Server(
					kratosotel.WithTracerProvider(tracerProvider),
					kratosotel.WithSchemaURL("https://opentelemetry.io/schemas/1.41.0"),
					kratosotel.WithAttributes(attribute.String("component", "kratos")),
				)),
			),
		),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

The instrumentation scope name is fixed to this package path. Use
`WithVersion`, `WithSchemaURL`, and `WithAttributes` to configure the
OpenTelemetry instrumentation scope metadata.

The middleware emits traces only. It does not create metrics instruments, so
histogram naming rules such as `_bucket` suffix handling are outside this
package. For `log/slog` to OpenTelemetry log records, use the official
`go.opentelemetry.io/contrib/bridges/otelslog` package instead of this
middleware.

## License

- The MIT License ([MIT](https://github.com/go-kratos-ecosystem/components/blob/2.x/LICENSE)). 
- [Kratos](https://github.com/go-kratos/kratos) License File: [License File](https://github.com/go-kratos/kratos/blob/8b8dc4b0f8bebb76939780f59734c20c265669c5/LICENSE)
