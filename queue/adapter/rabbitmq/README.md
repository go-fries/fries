# RabbitMQ Queue

A RabbitMQ AMQP 0.9.1 queue implementation for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/rabbitmq/v3
```

## Usage

```go
package main

import (
	"context"

	"github.com/go-fries/fries/queue/v3"
	"github.com/go-fries/fries/queue/adapter/rabbitmq/v3"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/")
	ch, _ := conn.Channel()

	q := rabbitmq.NewQueue(
		ch,
		rabbitmq.WithPrefix("app"),
	)
	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		queue.Handle("send_email", queue.HandlerFunc(func(ctx context.Context, task *queue.Task) error {
			// process task
			return nil
		})),
	)

	_, _ = producer.Enqueue(context.Background(), "send_email", []byte("hello"))
	_ = worker.Run(context.Background())
}
```

The adapter declares durable ready queues by default. Queue names map directly
to RabbitMQ queue names unless `WithPrefix` is used; for example, prefix `app`
stores queue `emails` as `app.emails`.

Delayed tasks are stored in RabbitMQ TTL queues that dead-letter back to the
ready queue after the delay, so the adapter does not require the RabbitMQ
delayed message exchange plugin. RabbitMQ re-delivers unacknowledged tasks when
the channel or connection closes; the queue visibility timeout passed to
`Dequeue` is not used by this adapter.
