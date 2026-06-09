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
	defer conn.Close()

	q := rabbitmq.NewQueue(
		conn,
		rabbitmq.WithPrefix("app"),
		rabbitmq.WithPrefetch(1),
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

## Publishing

Publisher confirms are enabled by default. `Enqueue`, retry publishing, and
dead-letter publishing wait until RabbitMQ acknowledges or negatively
acknowledges the publish, or until the call's `context.Context` is canceled.
Use a context deadline to bound publish latency. Use `WithPublisherConfirm(false)`
only when the application intentionally prefers lower publish latency over a
broker persistence acknowledgment.

The adapter opens a short-lived AMQP channel for each publish operation. This
keeps channels from being shared across goroutines and makes each publish
confirmation correspond to a single message.

## Delayed Tasks and Retries

Delayed tasks are stored in RabbitMQ TTL queues that dead-letter back to the
ready queue after the delay, so the adapter does not require the RabbitMQ
delayed message exchange plugin. The current implementation creates one delay
queue per queue name and delay value, for example `emails.delay.1500`, so prefer
bounded retry delays in production workloads. If a retry policy can produce many
distinct delays, bucket those delays in the policy before handing them to the
worker.

`Delivery.Retry` republishes a cloned task with the requested delay and then
acknowledges the original delivery. It does not use RabbitMQ `Nack` requeue
because the queue component's retry contract needs to preserve task metadata,
attempt state, and delayed retry behavior. If the retry publish succeeds but the
original ack fails, RabbitMQ may redeliver the original message, so handlers
must be idempotent.

## Dead Letters

`Delivery.DeadLetter` publishes the task to an adapter-managed queue named
`<queue>.dead` and then acknowledges the original delivery. The reason is stored
in task metadata and in the AMQP message headers. The adapter does not currently
configure a broker-native DLX for handler failures; the TTL delay queues use
RabbitMQ dead-lettering internally only to move delayed tasks back to the ready
queue.

## Consumers and Connections

RabbitMQ re-delivers unacknowledged tasks when the channel or connection closes.

The adapter opens AMQP channels internally. Producers use short-lived channels
for publish operations, and each consumer owns one channel for receiving and
acknowledging deliveries. RabbitMQ channels are not safe to share between
goroutines, so applications should share the connection and let the adapter
manage channels.

Consumers set RabbitMQ QoS prefetch to `1` by default so a consumer does not
reserve more unacknowledged deliveries than it can process. Use `WithPrefetch`
to tune this per-consumer limit, or set it to `0` to use RabbitMQ's unlimited
prefetch behavior.

The underlying `github.com/rabbitmq/amqp091-go` client does not automatically
reconnect a closed connection or channel. When the connection is closed, publish
and receive operations return errors; applications should recreate the
connection and queue adapter outside this package.
