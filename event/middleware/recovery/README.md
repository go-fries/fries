# Event Recovery Middleware

Recovery converts panics raised by event handlers into structured
`*recovery.PanicError` values. It does not log, emit metrics, or report alerts.
Those policies belong to outer middleware or the dispatch caller.

## Installation

```bash
go get github.com/go-fries/fries/event/middleware/recovery/v4
```

## Usage

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-fries/fries/event/middleware/recovery/v4"
	"github.com/go-fries/fries/event/v4"
)

type Event struct{}

func main() {
	dispatcher := event.New(
		event.WithMiddleware(recovery.New()),
	)
	dispatcher.Subscribe(
		event.HandlerFor[Event](event.HandlerFunc[Event](
			func(context.Context, Event) error {
				panic("boom")
			},
		)),
	)

	err := dispatcher.Dispatch(context.Background(), Event{})
	var panicError *recovery.PanicError
	if errors.As(err, &panicError) {
		fmt.Printf("panic: %v\n%s", panicError.Value, panicError.Stack)
	}
}
```

When the recovered value is an error, `PanicError.Unwrap` exposes it to
`errors.Is` and `errors.As`. Use `recovery.WithStackSize` to adjust the stack
buffer when necessary.
