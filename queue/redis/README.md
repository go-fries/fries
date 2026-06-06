# Redis Queue

A Redis Streams queue implementation for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/redis/v3
```

## Usage

```go
package main

import (
	"context"

	"github.com/go-fries/fries/queue/v3"
	queueredis "github.com/go-fries/fries/queue/redis/v3"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	q := queueredis.NewQueue(
		client,
		queueredis.WithPrefix("app"),
		queueredis.WithGroup("workers"),
		queueredis.WithConsumer("worker-1"),
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

The queue stores ready tasks in Redis Streams, delayed tasks in a sorted set,
and exhausted tasks in a dead-letter stream. Consumer groups are created lazily
with `XGROUP CREATE ... MKSTREAM`.
