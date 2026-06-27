# Queue Kratos Server

Kratos server adapter for `github.com/go-fries/fries/queue/v4` workers.

## Installation

```bash
go get github.com/go-fries/fries/queue/kratos/server/v4
```

## Usage

```go
package main

import (
	server "github.com/go-fries/fries/queue/kratos/server/v4"
	"github.com/go-fries/fries/queue/v4"
	"github.com/go-kratos/kratos/v2"
)

func newApp(q queue.Queue, handler queue.Handler) *kratos.App {
	worker := queue.NewWorker(
		q,
		queue.Handle("send_email", handler),
	)

	return kratos.New(
		kratos.Server(server.New(worker)),
	)
}
```

`Start` runs the worker and blocks until it exits. `Stop` delegates to
`Worker.Stop(ctx)`: it stops polling for new tasks and waits for in-flight
handlers. If the Kratos stop context expires first, the worker cancels running
handler contexts and returns the stop context error.
