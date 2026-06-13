# Crontab

`crontab` adapts [`go-cron`](https://github.com/flc1125/go-cron) schedulers to the
Kratos server lifecycle.

The package does not wrap cron's scheduling API. Configure jobs, parsers,
locations, middleware, and loggers on `*cron.Cron`, then pass that scheduler to
`crontab.NewServer`.

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
	c := cron.New(cron.WithSeconds())
	server := crontab.NewServer(c)

	_, err := server.Cron().AddFunc("* * * * * *", func(context.Context) error {
		// do something
		return nil
	})
	if err != nil {
		panic(err)
	}

	app := kratos.New(
		kratos.Server(server),
	)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
```

## Server Lifecycle

`NewServer` expects a configured `*cron.Cron`.

```go
c := cron.New(
	cron.WithSeconds(),
	cron.WithMiddleware(/* ... */),
)
server := crontab.NewServer(c)
```

`Start` runs the scheduler and blocks until `Stop` is called. `Stop` stops the
scheduler and waits for running jobs to finish. If the stop context is canceled
before jobs finish, `Stop` returns `ctx.Err()`.

Use `Server.Cron()` when jobs need to be registered after the server is created:

```go
_, err := server.Cron().AddFunc("@every 1m", func(context.Context) error {
	return nil
})
```

## Logging

`NewLogger` adapts a Kratos logger to cron's `Printf` logger shape. Use
`cron.PrintfLogger` or `cron.VerbosePrintfLogger` to pass it to go-cron.

```go
c := cron.New(
	cron.WithLogger(
		cron.VerbosePrintfLogger(
			crontab.NewLogger(crontab.WithLogger(log.DefaultLogger)),
		),
	),
)
```

`NewLogger()` uses Kratos' default logger. `WithLogger(nil)` leaves the current
logger unchanged.
