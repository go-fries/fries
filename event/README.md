# Event Dispatcher

The `Dispatcher` is a simple event dispatcher that allows you to dispatch events to registered listeners.

## Installation

```bash
github.com/go-fries/fries/event/v3
```

## Usage

```go
package event_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/event/v3"
)

func Example() {
	dispatcher := event.NewDispatcher()

	dispatcher.RegisterListeners(
		event.AdaptListener(event.ListenerFunc[*UserEvent](func(ctx context.Context, event *UserEvent) error {
			fmt.Println("this is user func listener, the name is", event.Name)
			return nil
		})),
		event.AdaptListener(&UserListener{}),
	)

	if err := dispatcher.Dispatch(context.Background(), &UserEvent{Name: "zhangsan"}); err != nil {
		fmt.Println(err)
	}
}

type UserEvent struct {
	Name string
}

type UserListener struct{}

func (u *UserListener) Handle(ctx context.Context, event *UserEvent) error {
	fmt.Println("this is user struct listener, the name is", event.Name)
	return nil
}
```