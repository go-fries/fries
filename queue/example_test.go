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

func ExampleDefine() {
	// Put the payload type and definition in a package shared by both services.
	type WelcomeEmail struct {
		UserID int `json:"user_id"`
	}
	welcomeEmail := queue.Define[WelcomeEmail]("notifications.welcome_email.v1")

	// Producer-side code only needs the shared definition and its Producer.
	enqueueWelcome := func(ctx context.Context, producer *queue.Producer, userID int) error {
		_, err := welcomeEmail.Enqueue(ctx, producer, WelcomeEmail{UserID: userID})
		return err
	}

	// Consumer-side code supplies its own backend and business dependencies.
	newWorker := func(backend queue.Queue, sendEmail func(context.Context, WelcomeEmail) error) *queue.Worker {
		return queue.NewWorker(backend, welcomeEmail.Handle(sendEmail))
	}
	_, _ = enqueueWelcome, newWorker
}
