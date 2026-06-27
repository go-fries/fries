# RabbitMQ Queue

RabbitMQ AMQP 0.9.1 adapter for `github.com/go-fries/fries/queue/v4`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/rabbitmq/v4
```

## Usage

```go
package main

import (
	rabbitmq "github.com/go-fries/fries/queue/adapter/rabbitmq/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

func newQueue() (*rabbitmq.Queue, func() error, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, nil, err
	}

	q := rabbitmq.NewQueue(
		conn,
		rabbitmq.WithPrefix("app"),
		rabbitmq.WithPrefetch(1),
	)
	return q, conn.Close, nil
}
```

Use `queue.NewProducer(q)` and `queue.NewWorker(q, ...)` from the core package
to enqueue and process tasks.

## Queues and Channels

Ready queues are durable by default. Queue names map directly to RabbitMQ queue
names unless `WithPrefix` is used.

The adapter opens AMQP channels internally. Publish operations use short-lived
channels, and each consumer owns one channel for receiving and acknowledging
deliveries. Applications should share the connection and let the adapter manage
channels.

Consumers set QoS prefetch to `1` by default. Use `WithPrefetch` to tune the
number of unacknowledged deliveries reserved by one consumer.

## Publishing

Publisher confirms are enabled by default. `Enqueue`, retry publishing, and
dead-letter publishing wait for RabbitMQ ack/nack or for the call context to be
canceled. Use a context deadline to bound publish latency.

Use `WithPublisherConfirm(false)` only when lower publish latency is more
important than waiting for broker confirmation.

## Delayed Retry and Dead Letter

Delayed tasks use RabbitMQ TTL queues that dead-letter back to the ready queue.
This does not require the RabbitMQ delayed message exchange plugin.

The adapter creates one TTL queue per queue name and delay value. Keep retry
delays bounded or bucketed in production.

`Delivery.Retry` republishes a cloned task and then acknowledges the original
delivery. It does not use `Nack(requeue=true)` because queue retries need to
preserve task metadata, attempt state, and delayed retry behavior.

`Delivery.DeadLetter` publishes to an adapter-managed `<queue>.dead` queue and
then acknowledges the original delivery. Broker-native DLX policies can still
be configured outside this package when your deployment needs central
RabbitMQ-side routing.

## Connections

`github.com/rabbitmq/amqp091-go` does not automatically reconnect closed
connections or channels. When the connection closes, publish and receive
operations return errors. Recreate the connection and queue adapter outside this
package.
