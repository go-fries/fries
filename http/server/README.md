# HTTP Server

A common HTTP server wrapper that can be used directly or as a Kratos-compatible server.

## Direct use

```go
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-fries/fries/http/server/v4"
)

func main() {
	srv := server.New(&http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		}),
	})

	if err := srv.Start(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
```

## Use with Kratos

```go
package main

import (
	"net/http"

	"github.com/go-fries/fries/http/server/v4"
	"github.com/go-kratos/kratos/v2"
)

func main() {
	srv := server.NewWithHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		}),
		server.WithAddr(":8080"),
	)

	app := kratos.New(
		kratos.Server(srv),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
```
