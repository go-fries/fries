# Queue

A durable task queue component for asynchronous background work.

## Installation

```bash
go get github.com/go-fries/fries/queue/v3
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/queue/adapter/memory/v3"
	"github.com/go-fries/fries/queue/v3"
)

func main() {
	ctx := context.Background()
	q := memory.NewQueue()

	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		queue.Handle("send_email", queue.HandlerFunc(func(ctx context.Context, task *queue.Task) error {
			fmt.Println(string(task.Payload))
			return nil
		})),
		queue.WithConcurrency(4),
	)

	_, _ = producer.Enqueue(ctx, "send_email", []byte("hello"), queue.WithQueue("default"))
	_ = worker.Run(ctx)
}
```

## Delivery Semantics

Tasks are delivered at least once. A handler may receive the same task again
after a crash, timeout, queue redelivery, or retry. Keep business idempotency
keys in the payload or metadata, and make handlers idempotent when duplicate
side effects matter.

## Typed Payloads

Use `EnqueueFor` and `HandleFor` when a task payload should be encoded and
decoded as a Go type. Typed helpers use JSON by default and keep the queue
payload as `[]byte`.

```go
type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

worker := queue.NewWorker(
	q,
	queue.HandleFor("send_email", queue.HandlerFuncFor[SendEmail](func(ctx context.Context, task *queue.TaskFor[SendEmail]) error {
		// task.Payload is SendEmail.
		// task.Task.Payload is the original []byte payload.
		return nil
	})),
)

_, _ = queue.EnqueueFor(ctx, producer, "send_email", SendEmail{UserID: 1, Subject: "welcome"})
```

## Retry and Dead Letter

Workers ACK tasks only when handlers return `nil`. Handler errors are retried
according to the configured retry policy. When a task exhausts its retry budget,
the queue moves it to a dead-letter queue.
