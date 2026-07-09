# EventBus

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/eventbus/v4"
)

type Event struct {
	ID int
}

type Listener struct{}

var _ eventbus.Handler[*Event] = (*Listener)(nil)

func NewListener() *Listener {
	return &Listener{}
}

func (l Listener) Handle(ctx context.Context, msg *Event) error {
	fmt.Println("Listener", msg.ID)
	return nil
}

func main() {
	event := eventbus.NewEvent[*Event]()

	// on with HandlerFunc
	listener := event.On(eventbus.HandlerFunc[*Event](func(_ context.Context, msg *Event) error {
		fmt.Println("HandlerFunc", msg.ID)
		return nil
	}))
	// on with Listener
	event.On(NewListener())

	// emit
	_ = event.Emit(context.Background(), &Event{2})
	// Output:
	// HandlerFunc 2
	// Listener 2

	// emit with async
	_ = event.EmitAsync(context.Background(), &Event{2})
	_ = event.Emit(context.Background(), &Event{2}, eventbus.WithEmitAsync())

	// emit with skip errors
	_ = event.Emit(context.Background(), &Event{2}, eventbus.WithEmitSkipErrors())

	// off listener
	_ = listener.Off()

	// off all listeners
	event.OffAll()

	// listeners
	event.Listeners()

	// listeners count
	event.ListenersCount()
}
```
