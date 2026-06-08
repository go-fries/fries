# Queue Kratos Server

Kratos server adapter for `github.com/go-fries/fries/queue/v3` workers.

## Installation

```bash
go get github.com/go-fries/fries/queue/kratos/server/v3
```

## Usage

```go
worker := queue.NewWorker(
	q,
	queue.HandleTasker[SendEmail](sendEmailTasker),
)

app := kratos.New(
	kratos.Server(server.New(worker)),
)
```

`Start` runs the worker and blocks until the worker exits. `Stop` cancels the
worker context and waits for it to return, so normal Kratos shutdown can stop
queue processing cleanly.
