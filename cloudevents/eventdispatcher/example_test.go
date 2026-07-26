package eventdispatcher_test

import (
	"context"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-fries/fries/cloudevents/eventdispatcher/v4"
)

type userCreated struct {
	UserID string `json:"user_id"`
}

func ExampleDispatcher() {
	dispatcher := eventdispatcher.NewDispatcher()
	dispatcher.AddListener(
		"user.created",
		eventdispatcher.ListenerFunc[userCreated](
			func(_ context.Context, event userCreated) error {
				fmt.Println("created user:", event.UserID)
				return nil
			},
		),
	)

	event := cloudevents.NewEvent()
	event.SetID("event-1")
	event.SetSource("example/users")
	event.SetType("user.created")
	if err := event.SetData(
		cloudevents.ApplicationJSON,
		userCreated{UserID: "123"},
	); err != nil {
		panic(err)
	}

	if err := dispatcher.Dispatch(context.Background(), event); err != nil {
		panic(err)
	}

	// Output:
	// created user: 123
}
