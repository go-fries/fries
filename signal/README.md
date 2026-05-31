# Signal Server

## Installation

```bash
go get github.com/go-fries/fries/signal/v3
```

## Example

```go
package main

import (
	"context"
	"os"
	"syscall"

	"github.com/go-kratos/kratos/v2"

	"github.com/go-fries/fries/signal/v3"
)

func main() {
	app := kratos.New(
		kratos.Server(newSignalServer()),
	)

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func newSignalServer() *signal.Server {
	srv := signal.NewServer(
		signal.WithHandlers(&exampleHandler{}, &example2Handler{}),
		signal.WithRecovery(signal.DefaultRecovery),
	)

	return srv
}

type exampleHandler struct{}

func (h *exampleHandler) Listen() []os.Signal {
	return []os.Signal{syscall.SIGUSR1, syscall.SIGUSR2}
}

func (h *exampleHandler) Handle(ctx context.Context, sig os.Signal) {
	println("exampleHandler signal:", sig)
}

type example2Handler struct {
	signal.AsyncHandler
}

func (h *example2Handler) Listen() []os.Signal {
	return []os.Signal{syscall.SIGUSR1}
}

func (h *example2Handler) Handle(context.Context, os.Signal) {
	panic("example2Handler panic")
}
```

Send signal:

```bash
$ kill -SIGUSR2 42750
$ kill -SIGUSR1 42750
```

Output:

```bash
INFO msg=[Signal] server starting
exampleHandler signal: (0x104ff0240,0x1051875b8)
exampleHandler signal: (0x104ff0240,0x1051875b0)
ERROR msg=[Signal] handler panic (user defined signal 1): example2Handler panic
```

## Behavior

- `Start` blocks until the context is canceled or `Stop` is called.
- `Stop` is idempotent and can be called more than once.
- `WithHandlers` registers handlers during construction. `Register` can add handlers before `Start` builds its signal routes.
- Handlers that embed `signal.AsyncHandler` run in their own goroutine.
- `WithRecovery` handles panics raised by handlers. Use `signal.WithRecovery(signal.DefaultRecovery)` to log handler panics.
