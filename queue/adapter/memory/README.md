# Memory Queue

An in-memory queue implementation for `github.com/go-fries/fries/queue/v3`.

## Installation

```bash
go get github.com/go-fries/fries/queue/adapter/memory/v3
```

## Usage

```go
package main

import (
	"context"

	"github.com/go-fries/fries/queue/adapter/memory/v3"
	"github.com/go-fries/fries/queue/v3"
)

func main() {
	q := memory.NewQueue()
	producer := queue.NewProducer(q)

	_, _ = producer.Enqueue(context.Background(), "send_email", []byte("hello"))
}
```

## Production Fit

The memory adapter is useful for tests, examples, and local development. It is
not a production adapter.

Tasks are not persisted across process restarts. A task is removed from the
in-memory queue before it is delivered to a handler, so an in-flight task is not
recovered if the process exits before the handler retries or dead-letters it.

The adapter does not simulate backend redelivery for crashed workers. It keeps
the implementation intentionally small so unit tests and examples stay
predictable.
