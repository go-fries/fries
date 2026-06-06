package queue_test

import (
	"context"
	"fmt"
	"time"

	"github.com/go-fries/fries/queue/v3"
)

func ExampleNewWorker() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	backend := queue.NewMemoryBackend()
	producer := queue.NewProducer(backend)
	worker := queue.NewWorker(
		backend,
		queue.Handle("send_email", queue.HandlerFunc(func(_ context.Context, task *queue.Task) error {
			fmt.Printf("%s: %s\n", task.Type, task.Payload)
			return nil
		})),
		queue.WithPollInterval(time.Millisecond),
	)

	if _, err := producer.Enqueue(ctx, "send_email", []byte("hello")); err != nil {
		panic(err)
	}
	if err := worker.Run(ctx); err != nil {
		panic(err)
	}

	// Output:
	// send_email: hello
}

type sendEmailPayload struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

func ExampleEnqueueFor() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	backend := queue.NewMemoryBackend()
	producer := queue.NewProducer(backend)
	worker := queue.NewWorker(
		backend,
		queue.HandleFor("send_email", queue.HandlerFuncFor[sendEmailPayload](func(_ context.Context, task *queue.TaskFor[sendEmailPayload]) error {
			fmt.Printf("%d: %s\n", task.Payload.UserID, task.Payload.Subject)
			return nil
		})),
		queue.WithPollInterval(time.Millisecond),
	)

	_, err := queue.EnqueueFor(ctx, producer, "send_email", sendEmailPayload{
		UserID:  100,
		Subject: "welcome",
	})
	if err != nil {
		panic(err)
	}
	if err := worker.Run(ctx); err != nil {
		panic(err)
	}

	// Output:
	// 100: welcome
}
