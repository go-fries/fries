package event_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/event/v4"
)

type orderPaid struct {
	orderID string
}

func ExampleDispatcher() {
	dispatcher := event.New()
	subscription := dispatcher.Subscribe(
		event.HandlerFor[orderPaid](event.HandlerFunc[orderPaid](
			func(_ context.Context, value orderPaid) error {
				fmt.Println("paid order:", value.orderID)
				return nil
			},
		)),
	)
	defer subscription.Unsubscribe()

	err := dispatcher.Dispatch(
		context.Background(),
		orderPaid{orderID: "123"},
	)
	fmt.Println("dispatch error:", err)

	// Output:
	// paid order: 123
	// dispatch error: <nil>
}
