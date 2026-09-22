# Queue

Backend-agnostic task queue primitives for asynchronous work in Go services.

The core package defines producers, workers, handlers, retry policies, and
delivery settlement. Storage and broker behavior are provided by adapters such
as Redis, RabbitMQ, and memory.

## Installation

```bash
go get github.com/go-fries/fries/queue/v4
```

Install the adapter you use as well:

```bash
go get github.com/go-fries/fries/queue/adapter/redis/v4
go get github.com/go-fries/fries/queue/adapter/rabbitmq/v4
go get github.com/go-fries/fries/queue/adapter/memory/v4
```

## Recommended usage

Create a backend, Producer and Worker during application setup. For business
payloads, start with [Typed Tasks](#typed-tasks): `EnqueueFor` encodes a value,
and `HandlePayload` registers a function that receives the decoded value. The
payload type is inferred from the function's argument. Use `HandleFor` with a
handler object or `HandlerFuncFor[T]` when delivery metadata is needed.

Define a stable task name once in application code and share it between
producer and consumer. `Tasker` and `HandleTasker` are useful when one object
should own the task name and its handling behavior. `TaskFor[T].Payload`
contains the decoded value; `TaskFor[T].Task` exposes delivery metadata.

Enqueueing returns a task and an error; successful enqueueing does not mean
the handler has run. Handlers return errors to the Worker's retry and settlement
policy. Pass producer/worker options during construction and enqueue options
for individual tasks, then arrange [graceful shutdown](#shutdown).

Use `HandlePayloadWithCodec` for a payload-only function with a custom codec.
See [Raw Tasks](#raw-tasks) for already encoded payloads and manual decoding.

## Basic Usage

### Typed Tasks

`EnqueueFor` and `HandlePayload` use JSON by default. Define the task name once
and use it for both sending and registration:

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/queue/adapter/memory/v4"
	"github.com/go-fries/fries/queue/v4"
)

type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

const sendEmailTaskType = "send_email"

func run(ctx context.Context) error {
	q := memory.NewQueue() // use Redis or RabbitMQ for durable production storage

	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		queue.HandlePayload(sendEmailTaskType, func(_ context.Context, payload SendEmail) error {
			fmt.Printf("send %s email to user %d\n", payload.Subject, payload.UserID)
			return nil
		}),
	)

	if _, err := queue.EnqueueFor(ctx, producer, sendEmailTaskType, SendEmail{
		UserID:  1,
		Subject: "welcome",
	}); err != nil {
		return err
	}

	return worker.Run(ctx)
}
```

`Worker.Run` blocks until the worker stops or a queue operation fails. In real
services, call `Worker.Stop(ctx)` during shutdown to stop receiving new tasks
and wait for in-flight handlers.

`HandlePayload` returns a Worker option and follows the existing registration
rules: an empty task name or nil function is ignored, and the last registered
handler for a name wins. Decode errors skip the payload function and go through
the same middleware and retry policy as handler errors. Handler control errors
such as `ErrDiscard`, `RetryAfter` and `DeadLetter` retain their usual meaning.

For a custom codec, use
`HandlePayloadWithCodec(taskType, codec, handler)` and pair it with
`EnqueueForWithCodec`. A nil codec selects JSON.

## Task Metadata

Use `HandleFor` when a handler needs both the decoded Payload and delivery
metadata. `TaskFor[T].Task` exposes fields such as ID, Attempt and Metadata:

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/queue/v4"
)

type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

func register(ctx context.Context, q queue.Queue, producer *queue.Producer) (*queue.Worker, error) {
	const taskType = "send_email"
	worker := queue.NewWorker(
		q,
		queue.HandleFor(taskType, queue.HandlerFuncFor[SendEmail](func(_ context.Context, task *queue.TaskFor[SendEmail]) error {
			fmt.Printf("task %s, attempt %d: email user %d\n", task.Task.ID, task.Task.Attempt, task.Payload.UserID)
			return nil
		})),
	)

	_, err := queue.EnqueueFor(ctx, producer, taskType, SendEmail{
		UserID:  1,
		Subject: "welcome",
	})
	return worker, err
}
```

Use `Tasker` and `HandleTasker` when one type should own both enqueueing and
handling for a task type. See [examples/tasker](examples/tasker) for a runnable
example.

## Raw Tasks

Use `Producer.Enqueue` and `Handle` for already encoded payloads or manual
decoding. Given an existing Producer, enqueue raw JSON with:

```go
_, err := producer.Enqueue(ctx, "send_email", []byte(`{"user_id":1,"subject":"welcome"}`))
```

Register a `Handler` or `HandlerFunc` with `Handle`; its `*Task` argument contains
the raw Payload and delivery metadata. Worker retry and settlement behavior is
the same for raw and typed handlers.

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
package main

import (
	"context"
	"errors"
	"time"

	"github.com/go-fries/fries/queue/v4"
)

func syncUserHandler(rateLimited, invalidPayload, alreadyHandled bool) queue.Handler {
	return queue.HandlerFunc(func(ctx context.Context, task *queue.Task) error {
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
	})
}
```

For production workloads, configure a bounded attempt limit and reuse the base
retry component for backoff:

```go
package main

import (
	"time"

	"github.com/go-fries/fries/queue/v4"
	"github.com/go-fries/fries/retry/v4"
)

func newWorker(q queue.Queue) *queue.Worker {
	return queue.NewWorker(
		q,
		queue.WithMaxAttempts(5),
		queue.WithBackoff(
			retry.Jitter(
				retry.Exponential(time.Second, time.Minute),
				250*time.Millisecond,
			),
		),
	)
}
```

`WithMaxAttempts` includes the initial delivery. `WithRetryIf` can additionally
filter ordinary handler errors using the task and error. An explicit
`RetryAfter` bypasses that predicate, but still respects the attempt limit.

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
package main

import (
	"context"

	"github.com/go-fries/fries/queue/v4"
)

func withObserver(q queue.Queue, handler queue.Handler) (*queue.Producer, *queue.Worker) {
	observer := queue.ObserverFunc(func(ctx context.Context, event queue.Event) context.Context {
		// Record metrics, logs, or spans. Return ctx unchanged, or return a
		// derived context to propagate values through later lifecycle events.
		return ctx
	})

	producer := queue.NewProducer(q, queue.WithObserver(observer))
	worker := queue.NewWorker(q, queue.WithObserver(observer), queue.Handle("send_email", handler))
	return producer, worker
}
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
