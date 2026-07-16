# Event

Event provides synchronous, type-aware dispatching for in-process application
events. Handlers are registered for exact Go types, and each dispatch has a
clear completion point and error result.

## Installation

```bash
go get github.com/go-fries/fries/event/v4
```

Install the optional panic recovery middleware separately:

```bash
go get github.com/go-fries/fries/event/middleware/recovery/v4
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-fries/fries/event/middleware/recovery/v4"
	"github.com/go-fries/fries/event/v4"
)

type OrderPaid struct {
	OrderID string
}

type ReceiptHandler struct{}

func (ReceiptHandler) Handle(_ context.Context, value OrderPaid) error {
	fmt.Println("send receipt for:", value.OrderID)
	return nil
}

func logErrors(next event.AnyHandler) event.AnyHandler {
	return func(ctx context.Context, value any) error {
		err := next(ctx, value)
		if err != nil {
			slog.ErrorContext(ctx, "event handler failed", "event", value, "error", err)
		}
		return err
	}
}

func main() {
	dispatcher := event.New(
		event.WithMiddleware(logErrors, recovery.New()),
	)

	subscription := dispatcher.Subscribe(
		event.HandlerFor[OrderPaid](ReceiptHandler{}),
		event.HandlerFor[OrderPaid](
			event.HandlerFunc[OrderPaid](func(_ context.Context, value OrderPaid) error {
				fmt.Println("paid order:", value.OrderID)
				return nil
			}),
		),
	)
	defer subscription.Unsubscribe()

	if err := dispatcher.Dispatch(
		context.Background(),
		OrderPaid{OrderID: "123"},
	); err != nil {
		panic(err)
	}
}
```

Middleware is declared from outermost to innermost. In the example,
`logErrors` can observe the structured error returned by `recovery.New()`.

## Execution behavior

- `Dispatch` runs matching handlers serially in registration order by default.
- The first handler error stops dispatch and is returned to the caller.
- `ContinueOnError()` runs the remaining handlers and joins their errors.
- `WithConcurrency(limit)` enables bounded concurrency for one dispatch and
  still waits for every started handler before returning.
- Context cancellation stops handlers that have not started; running handlers
  receive the canceled context and remain responsible for observing it.
- `OrderPaid` and `*OrderPaid` are distinct event types. Interface-based
  subscriptions are not supported.
- A `Subscription` can own registrations for multiple event types and removes
  them together with `Unsubscribe`.

Event does not provide fire-and-forget delivery, persistence, retries, or
message settlement. Use `parallel` for managed in-process background work and
`queue` for reliable asynchronous delivery.
