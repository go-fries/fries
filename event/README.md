# Event Component

A strongly-typed event dispatcher with middleware support, allowing you to decouple application logic through event listeners.

## Installation

```bash
go get github.com/go-fries/fries/event/v3
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/event/middleware/recovery/v3"
	"github.com/go-fries/fries/event/v3"
)

type UserEvent struct {
	Name string
}

type UserListener struct{}

func (u *UserListener) Handle(_ context.Context, event *UserEvent) error {
	fmt.Printf("Struct Listener: %s\n", event.Name)
	return nil
}

func main() {
	// Create dispatcher
	dispatcher := event.NewDispatcher()

	// Use middleware (e.g., panic recovery)
	dispatcher.Use(
		recovery.New(),
	)

	// Register listeners
	dispatcher.RegisterListeners(
		// Functional listener
		event.AdaptListenerFunc(func(_ context.Context, event *UserEvent) error {
			fmt.Printf("Func Listener: %s\n", event.Name)
			return nil
		}),
		// Struct listener
		event.AdaptListener(&UserListener{}),
	)

	// Dispatch event
	if err := dispatcher.Dispatch(context.Background(), &UserEvent{Name: "ZhangSan"}); err != nil {
		fmt.Println(err)
	}

	// Wait for async listeners (if any)
	dispatcher.Wait()
}
```