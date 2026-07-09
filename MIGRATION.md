# Migrating to Fries 4.x

Fries 4.x is the line that aligns this repository with Kratos v3 and standard
library `log/slog`. It is not source-compatible with every 3.x import path.

## Import Paths

Update Fries imports from `/v3` to `/v4`:

```go
import "github.com/go-fries/fries/cache/v4"
```

If code still imports the original components repository, replace it with the
matching Fries module:

```go
// before
import "github.com/go-kratos-ecosystem/components/v2/eventbus"

// after
import "github.com/go-fries/fries/eventbus/v4"
```

The common interface module was renamed:

```go
// before
import "github.com/go-fries/fries/contract/v4"

// after
import "github.com/go-fries/fries/capability/v4"
```

## Kratos

Fries 4.x targets Kratos v3. Update Kratos imports from `/v2` to `/v3`:

```go
import "github.com/go-kratos/kratos/v3"
```

Kratos wrapper packages in this repository keep the lightweight `Start(ctx)` and
`Stop(ctx)` server shape, without reintroducing a Kratos dependency where the
component can remain independent.

## Logging

Fries 4.x uses standard library `log/slog` for component lifecycle logging.
Options named `WithLogger` now accept `*slog.Logger` in migrated components.

Kratos-specific log modules were removed. Use the `slog.Handler` replacements:

| Removed module | Replacement |
| --- | --- |
| `github.com/go-fries/fries/kratos/log/multi/v4` | `github.com/go-fries/fries/log/slog/multi/v4` |
| `github.com/go-fries/fries/kratos/log/syslog/v4` | `github.com/go-fries/fries/log/slog/syslog/v4` |
| `github.com/go-fries/fries/kratos/log/otel/v4` | `go.opentelemetry.io/contrib/bridges/otelslog` |

Example:

```go
handler := slogmulti.NewHandler(
	slog.NewTextHandler(os.Stdout, nil),
	slog.NewJSONHandler(os.Stderr, nil),
)
logger := slog.New(handler)
```

## OpenTelemetry

`kratos/middleware/otel` remains tracing middleware only. It does not provide a
`log/slog` to OpenTelemetry logs bridge. Use the official `otelslog` bridge for
logs:

```go
import "go.opentelemetry.io/contrib/bridges/otelslog"
```

## Validation

`kratos/middleware/protovalidate` keeps focusing on Buf protovalidate for
`proto.Message` requests. It does not replace Kratos v3
`middleware/validate.Validator` for generic `Validate() error` request types.

## Checklist

- Replace `github.com/go-kratos-ecosystem/components/v2/...` imports with
  `github.com/go-fries/fries/.../v4`.
- Replace Fries `/v3` imports with `/v4`.
- Replace Kratos `/v2` imports with `/v3`.
- Replace Kratos log components with `log/slog/*` or official `otelslog`.
- Replace `contract/v4` imports with `capability/v4`.
- Run module-level tests and `make lint/<module>` for changed modules.
