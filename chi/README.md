# go-chi server

- https://github.com/go-chi/chi

## Example

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-kratos/kratos/v2"

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
