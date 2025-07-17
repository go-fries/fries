# go-chi server

- https://github.com/go-chi/chi

## Example

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-kratos/kratos/v2"

	chis "github.com/go-fries/fries/v3/chi"
)

func main() {
	cs := chis.NewServer(
		chi.NewRouter(),
		chis.Addr(":8001"),
	)

	cs.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	app := kratos.New(
		kratos.Server(cs),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
```