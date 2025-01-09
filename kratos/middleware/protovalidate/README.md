# Proto-Validate

Proto-Validate is a middleware for [Kratos](https://github.com/go-kratos/kratos).

The protovalidate uses the [protovalidate-go](https://github.com/bufbuild/protovalidate-go) library to validate the request messages of the gRPC service.


## Usage Example

```go
package main

import (
	"log"

	"github.com/bufbuild/protovalidate-go"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/transport/http"
	
	middlewareprotovalidate "github.com/go-kratos-ecosystem/components/v2/middleware/protovalidate"
)

func main() {
	validator, err := protovalidate.New(
		protovalidate.WithFailFast(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	app := kratos.New(
		http.NewServer(
			http.Address(":8000"),
			middlewareprotovalidate.Server(
				middlewareprotovalidate.Validator(validator),
			),
		),
	)

	app.Run()
}
```