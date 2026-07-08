# Chi Server

- https://github.com/go-chi/chi

`chi` wraps a configured `*chi.Mux` as a Kratos-compatible server.

## Direct use

```go
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	chis "github.com/go-fries/fries/chi/v4"
)

func main() {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	cs := chis.NewServer(
		router,
		chis.WithAddr(":8001"),
	)

	if err := cs.Start(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
```

## Use with Kratos

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-kratos/kratos/v3"

	chis "github.com/go-fries/fries/chi/v4"
)

func main() {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	cs := chis.NewServer(
		router,
		chis.WithAddr(":8001"),
	)

	app := kratos.New(
		kratos.Server(cs),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

Configure routes and middleware on `router` before passing it to `NewServer`.
