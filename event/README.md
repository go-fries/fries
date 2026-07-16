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

<details>
<summary>Organizing events and handlers</summary>

Keep event code with its business area. Shared top-level directories for every
event and handler quickly become difficult to navigate:

```text
internal/
├── order/
│   ├── service.go
│   └── events.go                 # defines order.Paid
├── notification/
│   └── send_paid_receipt.go      # handles order.Paid
├── analytics/
│   └── record_order_revenue.go   # handles order.Paid
└── app/
    └── events.go                 # registers handlers
```

Define an event in the package that produces it. Keep each handler in the
package that owns the work it performs. Register subscriptions during
application startup so the wiring remains visible in one place:

```go
func registerEventHandlers(
	dispatcher *event.Dispatcher,
	sendReceipt *notification.SendPaidReceipt,
	recordRevenue *analytics.RecordOrderRevenue,
) *event.Subscription {
	return dispatcher.Subscribe(
		event.HandlerFor[order.Paid](sendReceipt),
		event.HandlerFor[order.Paid](recordRevenue),
	)
}
```

Name handlers after the work they perform, such as `SendPaidReceipt`, rather
than using generic names such as `OrderPaidHandler`. Application services
should usually receive a `*event.Dispatcher` explicitly. Package-level dispatch
is better suited to small applications and integration code.

</details>

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

## Default dispatcher

For small applications or package-level integration, `Subscribe` and
`Dispatch` delegate to a replaceable default Dispatcher:

```go
subscription := event.Subscribe(
	event.HandlerFor[OrderPaid](ReceiptHandler{}),
)
defer subscription.Unsubscribe()

if err := event.Dispatch(ctx, OrderPaid{OrderID: "123"}); err != nil {
	return err
}
```

Configure the default Dispatcher during application startup, before registering
listeners:

```go
event.SetDefault(
	event.New(
		event.WithMiddleware(logErrors, recovery.New()),
	),
)
```

Replacing the default Dispatcher does not migrate existing subscriptions.
Libraries should prefer an explicitly injected Dispatcher and should not
register global listeners from `init` functions.

When used with `lifecycle`, `NewProvider(dispatcher)` installs the Dispatcher
both as the package default and in the lifecycle Context during Bootstrap.
Shutdown does not restore the previous default because application shutdown is
treated as process termination.
