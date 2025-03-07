package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/event/middleware/recovery/v3"
	"github.com/go-fries/fries/event/v3"
)

func main() {
	dispatcher := event.NewDispatcher(event.WithoutError(), event.WithParallel(1))

	// Use middleware
	dispatcher.Use(
		recovery.New(),
	)

	dispatcher.RegisterListeners(
		event.AdaptListener(event.ListenerFunc[*UserEvent](func(_ context.Context, event *UserEvent) error {
			fmt.Println("this is user func listener, the name is", event.Name)
			return nil
		})),
		event.AdaptListenerFunc(func(_ context.Context, event *UserEvent) error {
			fmt.Println("this is user func listener, the name is", event.Name)
			return nil
		}),
		event.AdaptListener(&UserListener{}),
	)

	if err := dispatcher.Dispatch(context.Background(), &UserEvent{Name: "zhangsan"}, event.WithDispatchWithError()); err != nil {
		fmt.Println(err)
	}

	// Wait for all listeners to finish processing
	dispatcher.Wait()
}

type UserEvent struct {
	Name string
}

type UserListener struct{}

func (u *UserListener) Handle(_ context.Context, event *UserEvent) error {
	fmt.Println("this is user struct listener, the name is", event.Name)
	return nil
}
