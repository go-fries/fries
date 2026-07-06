# Kratos log components

Kratos-specific log components were removed from `4.x`. Kratos v3 uses
`log/slog`, so reusable logging components now live under `log/slog/*`.

## Migration

| Removed module | Replacement |
| --- | --- |
| `github.com/go-fries/fries/kratos/log/multi/v4` | `github.com/go-fries/fries/log/slog/multi/v4` |
| `github.com/go-fries/fries/kratos/log/syslog/v4` | `github.com/go-fries/fries/log/slog/syslog/v4` |
| `github.com/go-fries/fries/kratos/log/otel/v4` | `go.opentelemetry.io/contrib/bridges/otelslog` |

Use the replacement `slog.Handler` with Kratos v3 logging APIs, such as
`github.com/go-kratos/kratos/v3/log.NewLogger`.
