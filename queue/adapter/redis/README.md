# Redis Queue

Redis Streams adapter for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/redis/v3
```

## Usage

```go
package main

import (
	"time"

	redis "github.com/go-fries/fries/queue/adapter/redis/v3"
	goredis "github.com/redis/go-redis/v9"
)

func newQueue() *redis.Queue {
	client := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:6379"})
	return redis.NewQueue(
		client,
		redis.WithPrefix("app"),
		redis.WithGroup("workers"),
		redis.WithConsumer("worker-1"),
		redis.WithClaimMinIdle(5*time.Minute),
	)
}
```

Use `queue.NewProducer(q)` and `queue.NewWorker(q, ...)` from the core package
to enqueue and process tasks.

## Storage Model

Ready tasks are stored in Redis Streams. Delayed tasks are stored in a sorted
set and promoted to the ready stream when due. Dead-lettered tasks are stored in
a separate Redis Stream.

Consumer groups are created lazily with `XGROUP CREATE ... MKSTREAM`.

## Delivery

The adapter follows the queue package's at-least-once contract. Retry and
dead-letter settlement write the new entry first, then acknowledge the original
stream message. If the process exits or `XACK` fails between those operations,
Redis may deliver the original task again.

`WithClaimMinIdle` enables pending message claiming during receive. Set it to
`0` to disable that behavior.

Malformed stream entries are acknowledged and discarded during receive, and the
receive call returns the parse error.

## Operations

The default Redis consumer name is process-unique. Use `WithConsumer` or
worker-level `queue.WithConsumerName` when a stable identity is useful for
operations and debugging.

Ready streams are not trimmed by the adapter because Redis consumer groups need
the stream entry to reclaim pending deliveries after a worker exits. Use
`WithDeadLetterMaxLen` to enable approximate trimming for dead-letter streams.
Monitor ready streams and delayed sorted sets for long-running workloads.

Connection pooling, reconnect behavior, and command retry behavior come from
the `go-redis` client passed to `NewQueue`.
