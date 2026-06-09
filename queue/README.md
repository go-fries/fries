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

## Worker Shutdown

`Run` blocks until the worker exits. For graceful shutdown, call
`Worker.Stop(ctx)`: it stops polling for new tasks, waits for in-flight task
handlers to finish, and cancels running handlers only if the stop context
expires before the worker exits. Canceling the context passed to `Run` interrupts
the worker immediately and should be reserved for forced shutdown paths.

When using Kratos, prefer `queue/kratos/server`; its `Stop` method delegates to
`Worker.Stop(ctx)` so application shutdown follows the same graceful drain
semantics as other Kratos servers.

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

Use `Tasker` and `HandleTasker` when one type should own both enqueueing and
handling for a task type. This keeps the task type string in one place. Name
application helpers `Enqueue` when they add tasks to the queue.

```go
const sendEmailTaskType = "send_email"

type SendEmailTasker struct {
	producer *queue.Producer
}

func (t *SendEmailTasker) TaskType() string {
	return sendEmailTaskType
}

func (t *SendEmailTasker) Enqueue(ctx context.Context, payload SendEmail, opts ...queue.EnqueueOption) (*queue.Task, error) {
	return queue.EnqueueFor(ctx, t.producer, t.TaskType(), payload, opts...)
}

func (t *SendEmailTasker) Handle(ctx context.Context, task *queue.TaskFor[SendEmail]) error {
	// task.Payload is SendEmail.
	return nil
}

tasker := &SendEmailTasker{producer: producer}
worker := queue.NewWorker(q, queue.HandleTasker[SendEmail](tasker))
_, _ = tasker.Enqueue(ctx, SendEmail{UserID: 1, Subject: "welcome"})
```

See [examples/tasker](examples/tasker) for a runnable Tasker example.

## Retry and Dead Letter

Workers ACK tasks only when handlers return `nil`. Handler errors are retried
according to the configured retry policy. When a task exhausts its retry budget,
the queue moves it to a dead-letter queue.

Handlers can return control errors for explicit business decisions:

```go
queue.Handle("sync_user", queue.HandlerFunc(func(ctx context.Context, task *queue.Task) error {
	switch {
	case rateLimited:
		return queue.RetryAfter(30 * time.Second)
	case invalidPayload:
		return queue.DeadLetter("invalid payload")
	case alreadyHandled:
		return queue.ErrDiscard
	default:
		return errors.New("temporary failure") // handled by the retry policy
	}
}))
```

Use `JitterRetry` to add bounded jitter to any retry policy:

```go
worker := queue.NewWorker(
	q,
	queue.WithRetryPolicy(queue.JitterRetry(
		queue.ExponentialRetry(5, time.Second, time.Minute),
		500*time.Millisecond,
	)),
)
```

## Observability

Use an `Observer` to connect queue events to logging, metrics, or tracing
without wrapping every handler manually. Observer events intentionally include
only low-sensitivity task fields such as ID, type, queue, and attempt. Payload
and metadata are not included.

```go
observer := queue.ObserverFunc(func(ctx context.Context, event queue.Event) {
	switch event.Kind {
	case queue.EventEnqueued:
		// record enqueue metric
	case queue.EventHandlerFailed:
		// record handler error
	case queue.EventTaskRetried:
		// record retry delay
	}
})

producer := queue.NewProducer(q, queue.WithProducerObserver(observer))
worker := queue.NewWorker(
	q,
	queue.WithWorkerObserver(observer),
	queue.Handle("send_email", handler),
)
```

The core package stays logger- and tracing-agnostic. OpenTelemetry integration,
if needed, should be provided by a separate package built on top of `Observer`.

## Production Use

Choose an adapter based on the durability and operations model you need:

| Adapter | Use for | Notes |
| --- | --- | --- |
| `adapter/memory` | tests, examples, local development | In-flight tasks are lost when the process exits. Do not use it as production storage. |
| `adapter/redis` | Redis-backed background jobs | Uses Redis Streams, consumer groups, delayed sorted sets, pending message claims, and optional stream trimming. |
| `adapter/rabbitmq` | RabbitMQ-backed background jobs | Uses durable queues, publisher confirms by default, per-consumer channels, QoS prefetch, and TTL delay queues. |

Queue processing is at-least-once. Production handlers should be idempotent and
should use a business idempotency key when external side effects matter. A task
can be delivered again after a worker crash, a backend redelivery, a settlement
failure, or a retry.

Use a bounded retry policy for production workloads. `ExponentialRetry` keeps
short failures cheap, `JitterRetry` spreads retry bursts, and `NoRetry` is useful
for tasks that should go directly to dead-letter storage when they fail.

```go
worker := queue.NewWorker(
	q,
	queue.WithConcurrency(8),
	queue.WithHandlerTimeout(30*time.Second),
	queue.WithSettlementTimeout(10*time.Second),
	queue.WithRetryPolicy(queue.JitterRetry(
		queue.ExponentialRetry(5, time.Second, time.Minute),
		500*time.Millisecond,
	)),
	queue.Handle("send_email", handler),
)
```

For graceful shutdown, let `Worker.Stop(ctx)` drain in-flight handlers. In a
Kratos application, use `queue/kratos/server` so the framework calls `Stop(ctx)`
with the same shutdown deadline as other servers. Canceling the context passed
to `Run` is the force-stop path and cancels running handlers immediately.

Keep observability low-sensitive by default. `Observer` events do not include
payload or metadata, and the recovery middleware default logger records only
task ID, type, queue, and attempt. Custom middleware or observers can opt into
more detail when the application owns the data-safety tradeoff.

Review the adapter README before production rollout:

- [memory](adapter/memory): storage is process-local and non-durable.
- [redis](adapter/redis): configure consumer identity, pending claim policy,
  and stream retention for long-running deployments.
- [rabbitmq](adapter/rabbitmq): configure connection recovery outside this
  package, prefetch for backpressure, and bounded retry delays to avoid
  unbounded delay queue growth.
