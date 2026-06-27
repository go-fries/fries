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

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/http"

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
					kratosotel.WithSchemaURL("https://opentelemetry.io/schemas/1.37.0"),
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

## License

- The MIT License ([MIT](https://github.com/go-kratos-ecosystem/components/blob/2.x/LICENSE)). 
- [Kratos](https://github.com/go-kratos/kratos) License File: [License File](https://github.com/go-kratos/kratos/blob/8b8dc4b0f8bebb76939780f59734c20c265669c5/LICENSE)
