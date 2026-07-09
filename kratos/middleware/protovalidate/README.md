# Proto-Validate

Proto-Validate is a middleware for [Kratos](https://github.com/go-kratos/kratos).

The protovalidate uses the [protovalidate-go](https://github.com/bufbuild/protovalidate-go) library to validate the request messages of the gRPC service.


## Usage Example

```go
package main

import (
	"log"

	"buf.build/go/protovalidate"
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport/http"

	middlewareprotovalidate "github.com/go-fries/fries/kratos/middleware/protovalidate/v4"
)

func main() {
	validator, err := protovalidate.New(
		protovalidate.WithFailFast(),
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
