# OpenTelemetry Middleware

This directory groups Kratos middleware built on OpenTelemetry.

## Layout

- `tracing/`: tracing middleware for Kratos clients and servers.
- `metrics/`: reserved for future metrics middleware.
- `logging/`: reserved for future log correlation or log-related middleware.
- `internal/`: reserved for shared OpenTelemetry helpers used by multiple signal
  packages, such as semantic convention attributes, transport metadata, and
  low-cardinality attribute builders.

## Design Notes

Signal-specific packages should keep their public API focused on that signal.
For example, tracing should expose span middleware and tracing options, while
future metrics middleware should expose metric instruments and metric-specific
options.

Shared helpers should move under `internal/` only when at least two signal
packages need them. Tracing-only helpers should stay inside `tracing/` until
there is a concrete cross-signal use case.
