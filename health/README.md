# Health

`health` runs named application health checks and exposes stable reports for
probes and operational endpoints. Checks run with a shared timeout and bounded
concurrency.

## Installation

```bash
go get github.com/go-fries/fries/health/v4
```

## Register checks

Use `CheckFunc` to adapt clients that already expose a Context-aware health
operation:

```go
func newReadiness(db *sql.DB) *health.Registry {
	readiness := health.New(
		health.WithTimeout(2*time.Second),
		health.WithConcurrency(4),
	)
	readiness.Register("database", health.CheckFunc(db.PingContext))
	return readiness
}
```

`Check` returns every result in registration order. A failed check does not
stop the other checks:

```go
report := readiness.Check(ctx)
if !report.Healthy() {
	for _, result := range report.Results {
		if result.Err != nil {
			log.Printf("%s is unhealthy: %v", result.Name, result.Err)
		}
	}
}
```

The timeout is a total budget for one `Check`, not a separate timeout for each
checker. A checker must observe the supplied Context while performing blocking
work. The registry cannot forcibly stop a checker that ignores cancellation.

## Liveness and readiness

Use separate registries because the two probes answer different questions:

```go
liveness := health.New()
readiness := newReadiness(db)

mux.Handle("/livez", health.Handler(liveness))
mux.Handle("/readyz", health.Handler(readiness))
```

Liveness should report whether the process itself can continue running. Avoid
including remote services whose temporary failure would only cause the
orchestrator to restart an otherwise healthy process.

Readiness should report whether the instance can currently accept work. It may
include critical dependencies such as a database, cache, or message broker.

The HTTP handler accepts `GET` and `HEAD`. It returns `200 OK` for a healthy
report and `503 Service Unavailable` when any check fails. Responses include
the overall status and each check's name, status, and duration.

Checker error messages are hidden by default because they may reveal internal
addresses or topology. A protected operational endpoint can expose them
explicitly:

```go
mux.Handle(
	"/internal/readyz",
	health.Handler(readiness, health.WithErrorDetails()),
)
```

Registries do not run background goroutines or cache previous reports. Register
checks during application assembly; dynamic removal is intentionally not part
of the initial API.
