# HTTP Server

A common HTTP service component that supports Kratos Server.

## Usage Example

```go
package server_test

import (
	"net/http"

	"github.com/go-kratos/kratos/v2"

	"github.com/go-kratos-ecosystem/components/v2/http/server"
)

func Example() {
	srv := server.New(&http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		}),
	})

	app := kratos.New(
		kratos.Server(srv),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

```