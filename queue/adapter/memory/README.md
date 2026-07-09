# Memory Queue

In-memory adapter for `github.com/go-fries/fries/queue/v4`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/memory/v4
```

## Usage

```go
package main

import (
	"context"

	"github.com/go-fries/fries/queue/adapter/memory/v4"
	"github.com/go-fries/fries/queue/v4"
)

func enqueue(ctx context.Context) error {
	q := memory.NewQueue()
	producer := queue.NewProducer(q)

	_, err := producer.Enqueue(ctx, "send_email", []byte("hello"))
	return err
}
```

## Semantics

This adapter is for tests, examples, and local development. It is not durable
and should not be used as production task storage.

Tasks are stored in process memory. Once a task is delivered, it is removed from
the ready queue. If the process exits before the handler retries or
dead-letters the task, that in-flight task is lost.

The adapter intentionally does not simulate backend redelivery after worker
crashes. Keeping it simple makes unit tests and examples predictable.
