# Queue Tasker Example

This example shows a typed tasker that owns both enqueueing and handling for one
task type.

Run it with:

```bash
go run .
```

The example uses the in-memory adapter so it can run without external services.
It is intended to demonstrate the `Tasker`, `HandleTasker`, and typed payload
helpers; use the Redis or RabbitMQ adapters for durable production storage.
