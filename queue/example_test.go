package queue_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/queue/v4"
)

func ExampleHandlePayload() {
	type SendEmail struct {
		UserID  int    `json:"user_id"`
		Subject string `json:"subject"`
	}

	// Supply the application's queue backend when constructing the worker.
	newEmailWorker := func(backend queue.Queue) *queue.Worker {
		return queue.NewWorker(backend,
			queue.HandlePayload("send_email", func(_ context.Context, payload SendEmail) error {
				fmt.Printf("send %s email to user %d\n", payload.Subject, payload.UserID)
				return nil
			}),
		)
	}
	_ = newEmailWorker
}
