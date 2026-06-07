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

The memory adapter is useful for tests, examples, and local development. It does
not persist tasks across process restarts.
