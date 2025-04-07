package main

import (
	"context"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/go-fries/fries/cloudevents/protocol/amqp091/v3"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close() //nolint:errcheck

	channel, err := conn.Channel()
	if err != nil {
		log.Fatal(channel)
	}
	defer channel.Close() //nolint:errcheck

	protocol, err := amqp091.NewProtocolFromConfig(&amqp091.Config{
		Channel:    channel,
		Exchange:   "cloudevents-exchange",
		RoutingKey: "cloudevents-routing-key",
		Queue:      "cloudevents-queue",
	})
	if err != nil {
		log.Fatal(err)
	}

	client, err := cloudevents.NewClient(protocol)
	if err != nil {
		log.Fatal(err)
	}

	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetSource("example/uri")
	event.SetType("example.type")
	if err := event.SetData(cloudevents.ApplicationJSON, map[string]string{"hello": "world"}); err != nil {
		log.Fatal(err)
	}

	go func() {
		_ = client.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) error {
			log.Printf("Received event: %v", event)
			return nil
		})
	}()

	if result := client.Send(context.Background(), event); cloudevents.IsUndelivered(result) {
		log.Printf("Failed to send: %v", result)
	} else {
		log.Printf("Sent: %v", event)
	}

	time.Sleep(1 * time.Second)
}
