# Crontab

A Cron job scheduler integration for [Kratos](https://github.com/go-kratos/kratos).

## Installation

```bash
go get github.com/go-fries/fries/crontab/v3
```

## Usage

```go
package main

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/go-fries/fries/crontab/v3"
	"github.com/go-kratos/kratos/v2"
)

func main() {
	c := cron.New(
		cron.WithSeconds(),
		// cron.WithMiddleware( ... ),
	)

	_, _ = c.AddFunc("* * * * * *", func(context.Context) error {
		// do something
		return nil
	})

	// kratos app start
	app := kratos.New(
		kratos.Server(
			crontab.NewServer(c),
		),
	)

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
```