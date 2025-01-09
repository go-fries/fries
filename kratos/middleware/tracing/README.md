# Tracing Middleware

The tracing middleware provides a way to trace the execution of a request through the application. It is based on the [OpenTelemetry](http://opentelemetry.io/) standard and can be used with any tracer that implements this standard.

The package is forked from [tracing](https://github.com/go-kratos/kratos/tree/8b8dc4b0f8bebb76939780f59734c20c265669c5/middleware/tracing) and optimized on this basis. Thanks to the original author for his contribution.

## Usage Example

```go
package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/go-kratos-ecosystem/components/v2/middleware/tracing"
)

func main() {
	app := kratos.New(
		kratos.Name("tracing"),
		kratos.Server(
			http.NewServer(
				http.Address(":8001"),
				http.Middleware(tracing.Server()),
			),
		),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

## License

- The MIT License ([MIT](https://github.com/go-kratos-ecosystem/components/blob/2.x/LICENSE)). 
- [Kratos](https://github.com/go-kratos/kratos) License File: [License File](https://github.com/go-kratos/kratos/blob/8b8dc4b0f8bebb76939780f59734c20c265669c5/LICENSE)