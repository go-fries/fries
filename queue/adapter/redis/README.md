# Redis Queue

A Redis Streams queue implementation for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/redis/v3
```

## Usage

```go
package main

import (
	"context"
	"time"

	"github.com/go-fries/fries/queue/v3"
	"github.com/go-fries/fries/queue/adapter/redis/v3"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	client := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	q := redis.NewQueue(
		client,
		redis.WithPrefix("app"),
		redis.WithGroup("workers"),
		redis.WithConsumer("worker-1"),
		redis.WithClaimMinIdle(5*time.Minute),
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

## Storage

The queue stores ready tasks in Redis Streams, delayed tasks in a sorted set,
and exhausted tasks in a dead-letter stream. Consumer groups are created lazily
with `XGROUP CREATE ... MKSTREAM`.

Delayed tasks are stored as unique sorted set members, so two delayed tasks with
the same payload are still promoted independently.

## Consumers

The adapter generates a process-unique default Redis consumer name. Use
`WithConsumer` or worker-level `queue.WithConsumerName` when you need a stable
consumer identity for operations and debugging.

`WithClaimMinIdle` controls how long a pending stream message must remain idle
before a consumer can claim it for redelivery. Set it to `0` to disable pending
message claims during receive.

## Retention

Ready and dead-letter streams are not trimmed by default. Use
`WithStreamMaxLen` and `WithDeadLetterMaxLen` to enable approximate Redis stream
trimming for long-running deployments.

## Delivery Semantics

Retry and dead-letter handling follows the queue package's at-least-once
contract. The adapter writes the retry or dead-letter entry before acknowledging
the original stream message. If the acknowledgement fails or the process exits
between those steps, Redis may deliver the original task again, so handlers must
remain idempotent.

Malformed Redis stream entries are acknowledged and discarded during receive,
and the receive call returns the parse error to the worker.
