# Queue

Backend-agnostic task queue primitives for asynchronous work in Go services.

The core package defines producers, workers, handlers, retry policies, and
delivery settlement. Storage and broker behavior are provided by adapters such
as Redis, RabbitMQ, and memory.

## Installation

```bash
go get github.com/go-fries/fries/queue/v3
```

Install the adapter you use as well:

```bash
go get github.com/go-fries/fries/queue/adapter/redis/v3
go get github.com/go-fries/fries/queue/adapter/rabbitmq/v3
go get github.com/go-fries/fries/queue/adapter/memory/v3
```

## Basic Usage

```go
q := memory.NewQueue() // use Redis or RabbitMQ for durable production storage

producer := queue.NewProducer(q)
worker := queue.NewWorker(
	q,
	queue.Handle("send_email", queue.HandlerFunc(func(ctx context.Context, task *queue.Task) error {
		// Decode task.Payload and perform the work.
		return nil
	})),
)

_, err := producer.Enqueue(ctx, "send_email", []byte(`{"user_id":1}`), queue.WithQueue("default"))
if err != nil {
	return err
}

go func() {
	_ = worker.Run(ctx)
}()
```

`Worker.Run` blocks until the worker stops or a queue operation fails. In real
services, call `Worker.Stop(ctx)` during shutdown to stop receiving new tasks
and wait for in-flight handlers.

## Typed Tasks

Use `EnqueueFor` and `HandleFor` when a task payload should be encoded and
decoded as a Go type. JSON is used by default.

```go
type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

worker := queue.NewWorker(
	q,
	queue.HandleFor("send_email", queue.HandlerFuncFor[SendEmail](func(ctx context.Context, task *queue.TaskFor[SendEmail]) error {
		return sendEmail(ctx, task.Payload.UserID, task.Payload.Subject)
	})),
)

_, err := queue.EnqueueFor(ctx, producer, "send_email", SendEmail{
	UserID:  1,
	Subject: "welcome",
})
```

Use `Tasker` and `HandleTasker` when one type should own both enqueueing and
handling for a task type. See [examples/tasker](examples/tasker) for a runnable
example.

## Delivery Semantics

Queue delivery is at least once. A task may be delivered again after a process
crash, backend redelivery, retry, or settlement failure. Handlers should be
idempotent when duplicate side effects matter.

When a handler returns `nil` or `ErrDiscard`, the worker acknowledges the
delivery. Other handler errors are passed to the configured retry policy. A
retry schedules another delivery attempt; when the retry budget is exhausted,
the worker dead-letters the task.

Handlers can return control errors for explicit decisions:

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
		return errors.New("temporary failure")
	}
}))
```

For production workloads, prefer bounded retry policies. `JitterRetry` can wrap
another policy to spread retry bursts.

## Shutdown

`Worker.Stop(ctx)` stops receiving new deliveries and waits for in-flight
handlers. If the stop context expires, the worker cancels running handler
contexts and returns the stop context error.

Canceling the context passed to `Run` is the force-stop path: it cancels
receiving and running handlers immediately.

For Kratos applications, use `queue/kratos/server` so the framework delegates
shutdown to `Worker.Stop(ctx)`.

## Observability

`Observer` lets producers and workers emit low-sensitivity events for metrics,
logging, or tracing. Observer events include task ID, type, queue, and attempt;
they intentionally omit task payload and metadata.

```go
observer := queue.ObserverFunc(func(ctx context.Context, event queue.Event) {
	// Record metrics, logs, or spans.
})

producer := queue.NewProducer(q, queue.WithProducerObserver(observer))
worker := queue.NewWorker(q, queue.WithWorkerObserver(observer), queue.Handle("send_email", handler))
```

The core package does not depend on a logger or tracing implementation.

## Adapters

| Adapter | Intended use |
| --- | --- |
| [memory](adapter/memory) | Tests, examples, and local development. Not durable. |
| [redis](adapter/redis) | Redis Streams backed queues with delayed tasks and dead-letter streams. |
| [rabbitmq](adapter/rabbitmq) | RabbitMQ backed queues with publisher confirms, prefetch, delayed retries, and dead-letter queues. |

Read the adapter README before production use. Backend-specific behavior such as
retention, connection recovery, delayed retry implementation, and dead-letter
storage lives in the adapter.
