# Queue Tasker Example

Typed tasker example for `github.com/go-fries/fries/queue/v4`.

```bash
go run .
```

The example keeps the task type string, enqueue helper, and handler on one
`SendEmailTasker` type. It uses the in-memory adapter so it can run without
external services.
