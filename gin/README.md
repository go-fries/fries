# Gin Server

`gin` wraps a configured `*gin.Engine` as a Kratos-compatible server.

## Direct use

```go
package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	gins "github.com/go-fries/fries/gin/v4"
)

func main() {
	engine := gin.Default()
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	gs := gins.NewServer(
		engine,
		gins.WithAddr(":8080"),
	)

	if err := gs.Start(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
```

## Use with Kratos

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/go-kratos/kratos/v3"

	gins "github.com/go-fries/fries/gin/v4"
)

func main() {
	engine := gin.Default()
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	gs := gins.NewServer(
		engine,
		gins.WithAddr(":8080"),
	)

	app := kratos.New(
		kratos.Server(gs),
	)

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
```

Configure routes and middleware on `engine` before passing it to `NewServer`.
