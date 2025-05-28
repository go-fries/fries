# Event-Dispatcher for CloudEvents

## Usage

```go
package main

import (
	"context"
	"log"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-fries/fries/cloudevents/eventdispatcher/v3"
)

type ExampleEvent struct {
	UserID string
}

type ExampleListener struct{}

func (l *ExampleListener) Handle(ctx context.Context, event *ExampleEvent) error {
	log.Printf("Received event for user: %s", event.UserID)
	return nil
}

func main() {
	client, err := cloudevents.NewClient(nil) // Replace nil with your protocol configuration if needed
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	dispatcher := eventdispatcher.NewDispatcher()

	// the method one:
	dispatcher.AddListener(
		"example.type",
		eventdispatcher.ListenerFunc[*ExampleEvent](func(ctx context.Context, event *ExampleEvent) error {
			log.Printf("Received event for user: %s", event.UserID)
			return nil
		}),
	)
	// the method two:
	dispatcher.AddListener(
		"example.type",
		eventdispatcher.ListenerAdapter(&ExampleListener{}),
	)
	
	_ = client.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) error {
		log.Printf("Received event: %v", event)
		return nil
	})
}
```