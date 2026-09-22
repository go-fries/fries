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
payloads, share a `queue.Define[T](name)` between producer and consumer code.
Its `Enqueue` method encodes a value, and `Handle` registers a function that
receives the decoded value. See [Typed Tasks](#typed-tasks) for a complete
in-memory example and [Shared Task Definitions](#shared-task-definitions) for
separate producer and consumer packages.

Use a definition's `HandleFor` with a handler object or `HandlerFuncFor[T]` when
delivery metadata is needed. `TaskFor[T].Payload` contains the decoded value;
`TaskFor[T].Task` exposes delivery metadata. `Tasker` and `HandleTasker` remain
useful when one object should own the task name and its handling behavior.

Enqueueing returns a task and an error; successful enqueueing does not mean
the handler has run. Handlers return errors to the Worker's retry and settlement
policy. Pass producer/worker options during construction and enqueue options
for individual tasks, then arrange [graceful shutdown](#shutdown).

Pass `queue.WithCodec(codec)` to `Define` to select a custom codec for both sides.
See [Raw Tasks](#raw-tasks) for already encoded payloads and manual decoding.

## Basic Usage

### Typed Tasks

Define the task name and payload type once. This complete program uses the
in-memory adapter, enqueues one task, waits for the handler and stops the Worker.
It uses channel synchronization rather than a sleep; the timeout bounds the run.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-fries/fries/queue/adapter/memory/v4"
	"github.com/go-fries/fries/queue/v4"
)

type SendEmail struct {
	UserID  int    `json:"user_id"`
	Subject string `json:"subject"`
}

var sendEmail = queue.Define[SendEmail]("notifications.send_email.v1")

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := memory.NewQueue() // use Redis or RabbitMQ for durable production storage

	handled := make(chan struct{})
	producer := queue.NewProducer(q)
	worker := queue.NewWorker(
		q,
		sendEmail.Handle(func(_ context.Context, payload SendEmail) error {
			fmt.Printf("send %s email to user %d\n", payload.Subject, payload.UserID)
			close(handled) // this example sends exactly one task
			return nil
		}),
	)

	if _, err := sendEmail.Enqueue(ctx, producer, SendEmail{
		UserID:  1,
		Subject: "welcome",
	}); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	select {
	case <-handled:
		// Stop waits for the handler and delivery settlement to finish.
		stopErr := worker.Stop(ctx)
		cancel()
		return errors.Join(stopErr, <-done)
	case err := <-done:
		return err
	case <-ctx.Done():
		cancel()
		<-done
		return ctx.Err()
	}
}
```

`Worker.Run` blocks until the worker stops or a queue operation fails. In real
services, call `Worker.Stop(ctx)` during shutdown to stop receiving new tasks
and wait for in-flight handlers.

## Shared Task Definitions

Place the payload and definition in a small application contract package:

```go
package tasks

import "github.com/go-fries/fries/queue/v4"

type WelcomeEmail struct {
	UserID int `json:"user_id"`
}

var SendWelcome = queue.Define[WelcomeEmail]("notifications.welcome_email.v1")
```

Producer code imports that package and supplies its Producer. Given `tasks`,
`ctx`, `producer` and `userID` from the application:

```go
task, err := tasks.SendWelcome.Enqueue(ctx, producer, tasks.WelcomeEmail{UserID: userID},
	queue.WithQueue("mail"),
	queue.WithID("welcome-42"),
	queue.WithDelay(time.Minute),
	queue.WithMetadataValue("tenant", "acme"),
)
```

Consumer code imports the same contract and supplies its service dependencies:

```go
worker := queue.NewWorker(backend,
	queue.WithQueue("mail"),
	tasks.SendWelcome.Handle(func(ctx context.Context, payload tasks.WelcomeEmail) error {
		return mailer.SendWelcome(ctx, payload.UserID)
	}),
)
```

The producer does not need the mailer or a consumer-side handler object.
`Define[T]` returns a `Definition[T]` value containing the task name and codec. It
holds no Producer or handler and can be reused with different instances.
Treat a shared package-level definition as fixed after initialization.

### Names, registration and codecs

- Names are explicit wire identifiers returned by `TaskType()`. They are not
  inferred from Go type names. Keep them stable across deployments and coordinate
  payload schema changes between producers and consumers.
- An empty name or zero-value `Definition[T]` returns `ErrInvalidTaskType` from
  `Enqueue` before encoding or invoking the Producer. Registration
  with an empty name is ignored.
- `Handle` ignores nil functions. `HandleFor` follows
  the existing `HandlerFor` rules: nil interfaces are ignored. As with existing
  handlers, an interface holding a typed nil is not a nil interface.
- Definitions share the Worker's existing handler map: the last non-ignored
  registration for a name wins, including registrations via the old helpers.
  Different payload types with the same name do not create separate routes.
- JSON is the default. Configure `queue.WithCodec(codec)` once on `Define`;
  `Enqueue`, `Handle` and `HandleFor` all use that codec. `WithCodec(nil)` selects
  JSON, and the last codec option wins. The codec instance is retained, not
  cloned, so it must support concurrent use when the definition is shared.
  The task envelope does not record the codec; definitions in separate services
  must still use compatible encodings.
- Enqueue options, Producer observer events, middleware, decoding errors, retry
  decisions and settlement all follow the existing Queue paths. Decode errors
  skip the handler. `ErrDiscard`, `RetryAfter` and `DeadLetter` retain their usual
  meaning. The stored task envelope is unchanged.

For an existing `customCodec`, define the encoding alongside the task name:

```go
var SendWelcome = queue.Define[WelcomeEmail](
	"notifications.welcome_email.v1",
	queue.WithCodec(customCodec),
)
```

Both sides then use the same `SendWelcome.Enqueue`, `SendWelcome.Handle` or
`SendWelcome.HandleFor` calls as with JSON. Lower-level per-call codec selection
remains available through `EnqueueForWithCodec`, `HandlePayloadWithCodec` and
`HandleForWithCodec`.

### Existing typed helpers and Tasker

For a one-off task, `EnqueueFor` and `HandlePayload` remain available. For example,
the same task previously required the name at both call sites:

```go
queue.EnqueueFor(ctx, producer, "send_email", SendEmail{UserID: 42})
queue.HandlePayload("send_email", func(ctx context.Context, payload SendEmail) error {
	return send(ctx, payload)
})
```

A shared `sendEmail := queue.Define[SendEmail]("send_email")` changes those calls
to `sendEmail.Enqueue(ctx, producer, payload)` and `sendEmail.Handle(handler)`;
the task name and payload type are tied together at the definition.

`Tasker` is still appropriate when a concrete object owns its task name, service
dependencies and typed handling behavior, possibly with a business-specific
enqueue method. Use a `Definition` when producer and consumer need a shared
contract without sharing that handler object. See [examples/tasker](examples/tasker)
for the object-owned style.

## Task Metadata

Use a definition's `HandleFor` when a handler needs both the decoded Payload and
delivery metadata. `TaskFor[T].Task` exposes fields such as ID, Attempt and Metadata:

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
	sendEmail := queue.Define[SendEmail]("send_email")
	worker := queue.NewWorker(
		q,
		sendEmail.HandleFor(queue.HandlerFuncFor[SendEmail](func(_ context.Context, task *queue.TaskFor[SendEmail]) error {
			fmt.Printf("task %s, attempt %d: email user %d\n", task.Task.ID, task.Task.Attempt, task.Payload.UserID)
			return nil
		})),
	)

	_, err := sendEmail.Enqueue(ctx, producer, SendEmail{
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
