# Context Valuer for slog

`ctxvaluer` adds context-aware attributes to a standard `slog.Handler`. Use the
ordinary `slog.Logger`, including `InfoContext`, `LogAttrs`, `With`, and
`WithGroup`. The module requires Go 1.26.0. The implementation uses only the
standard library; tests use `github.com/stretchr/testify` for assertions.

## Installation

```sh
go get github.com/go-fries/fries/log/slog/ctxvaluer/v4
```

## Usage

Register shared attributes once. Each callback receives the context of the
current log call, not a context captured when the logger was constructed.

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
)

type userIDKey struct{}

func userID(ctx context.Context) slog.Value {
	id, ok := ctx.Value(userIDKey{}).(int64)
	if !ok {
		return ctxvaluer.Omit()
	}
	return slog.Int64Value(id)
}

func main() {
	logger := slog.New(ctxvaluer.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil),
		ctxvaluer.WithValue("user_id", userID),
	)).WithGroup("order").With("source", "marketplace")

	ctx := context.WithValue(context.Background(), userIDKey{}, int64(123))
	logger.InfoContext(ctx, "created", "order_id", 1001)
}
```

Output, excluding the timestamp:

```json
{"level":"INFO","msg":"created","user_id":123,"order":{"source":"marketplace","order_id":1001}}
```

See [`example_test.go`](example_test.go) for a runnable `net/http` middleware
example. It reads `X-Request-ID`, places it in the request context, and passes
`r.Context()` to the shared logger. No framework integration is required. Trace
and span IDs can be registered the same way; their extraction belongs in the
application or an integration package, not this module.

## API and field behavior

```go
type Valuer func(context.Context) slog.Value

func NewHandler(next slog.Handler, opts ...Option) *Handler
func WithValue(key string, value Valuer) Option
func WithCurrentGroup() Option
func Omit() slog.Value
```

`WithValue` follows the repository's `WithXxx` option naming convention. Values
use standard `slog.StringValue`, `Int64Value`, `AnyValue`, etc.; there are no
parallel conversion helpers or custom logger APIs.

| Situation | Behavior |
| --- | --- |
| Evaluation | Once per `Handle`, in registration order, before the child handler runs. `Enabled`, `WithAttrs`, and `WithGroup` do not execute Context Valuers. |
| Filtered event | `slog.Logger` does not call `Handle`; no Valuer executes. Direct `Handle` calls assume the caller has checked `Enabled`, as in native slog. |
| `0`, `false`, `""` | Retained. |
| Missing field | Return `Omit()`, an empty `slog.GroupValue()`. |
| Zero `slog.Value` | A nil value, normally rendered as JSON `null`; it is not an omission sentinel. An empty key with a nil value is the zero Attr and follows slog's omission rule. |
| Group value | Passed to the child handler; nested attributes are retained. An empty group is omitted; an empty group key inlines its contents. |
| Duplicate keys | Retained, including repeated registrations, bound fields, record fields, and collisions with built-in keys. No overwrite or deduplication guarantee. |
| Field order | The child's existing and root-bound fields precede registered dynamic fields. Dynamic fields precede record fields. Within a group, bound fields precede record fields. The child can transform or reorder output. |
| Nil context | Standard Logger methods substitute `context.Background()`. Use Logger methods rather than calling Handler methods with nil contexts. |
| Canceled context | Still passed through and evaluated; cancellation does not suppress logging. |
| Valuer panic | Propagates to the caller; this module does not recover it. Native `slog.LogValuer` resolution remains the child handler's responsibility. |
| Handler error | Returned unchanged by `Handle`. Standard `slog.Logger` discards handler errors; use a child error-reporting handler if required. |
| Invalid constructor arguments | A nil handler or registered Valuer panics during construction. |

Callback functions and the wrapped handler must support concurrent use. The
wrapper keeps no request context, changes no configuration after construction,
and starts no goroutines. It builds a new `slog.Record` when adding attributes,
preserving time, level, message, and source PC without changing the input record.
As with slog itself, do not mutate slices supplied to `slog.GroupValue` or shared
application values after handing them to a logger.

## Groups: two explicit modes

### Default: context fields outside wrapper-owned groups

Native `WithGroup` affects **all subsequent attributes**, including attributes
added to a record by an outer handler. Delegating `WithGroup` and then injecting
context attributes in `Handle` cannot keep those attributes at the top level.
The Handler interface provides no operation to leave a group or inspect its
current path.

The default mode therefore leaves the supplied handler ungrouped and owns the
groups created on the wrapper:

- Root `WithAttrs` calls delegate to the supplied handler.
- `WithGroup` records an immutable group path instead of delegating it.
- Grouped `WithAttrs` retains ordinary attributes at their position in that path.
- `Handle` builds **one nested GroupValue tree** from the retained attributes and
  the record attributes, after the context fields.

One tree matters: separately emitting `order` groups for bound and record
attributes would create duplicate JSON objects rather than one combined group.
The implementation uses only public slog APIs, never output rewriting or
JSONHandler internals. JSONHandler and TextHandler preserve normal attribute
grouping, including multi-level inheritance, inline groups, empty groups, and
`ReplaceAttr` group paths.

### Native delegation: context fields in the current group

Select this mode when the child handler needs its own `WithGroup` behavior or
when context fields should follow the logger's current group. Using `userID` and
`ctx` from the example above:

```go
handler := ctxvaluer.NewHandler(
	slog.NewJSONHandler(os.Stdout, nil),
	ctxvaluer.WithCurrentGroup(),
	ctxvaluer.WithValue("user_id", userID),
)
logger := slog.New(handler).WithGroup("order").With("source", "marketplace")
logger.InfoContext(ctx, "created", "order_id", 1001)
```

Here the output contains:

```json
{"order":{"source":"marketplace","user_id":123,"order_id":1001}}
```

Both `WithAttrs` and `WithGroup` delegate to the child. Dynamic values are still
computed during `Handle`, never during binding. This option applies to all
registered values; it does not mix global and group-local registrations.

### Compatibility limits

- To obtain output-level top-level context fields, wrap a handler **before** it
  has groups. Groups already on the supplied handler cannot be undone; the
  wrapper's root is relative to that handler.
- Default mode requires a child that handles `slog.GroupValue` attributes. It
  cannot reproduce third-party behavior triggered specifically by native
  `WithGroup` or grouped `WithAttrs`, such as group-dependent enablement or custom
  preformatting. Use `WithCurrentGroup()` for those handlers.
- Grouped bound attributes in default mode are delivered during `Handle`.
  Ordinary `slog.LogValuer` values and `ReplaceAttr` callbacks for those attributes
  can run on every event, instead of being preformatted once at binding time by
  JSONHandler or TextHandler. Keep them concurrency-safe and side-effect-free,
  or select native delegation when binding-time processing matters.
- This component is a registration-based injector. Passing a `ctxvaluer.Valuer`
  directly as an ordinary `logger.With` or log-call value does not register it.
- It does not merge duplicate fields or promise compatibility with arbitrary
  third-party grouping policies. JSON key order is not an overwrite policy.

## Validation and performance

From the repository root:

```sh
make test/log/slog/ctxvaluer
make test/log/slog/ctxvaluer ARGS='-race -count=1'
make lint/log/slog/ctxvaluer
```

From this module directory:

```sh
go test ./...
go test -race ./...
go vet ./...
go test -run '^$' -bench BenchmarkHandler -benchmem -count=5
```

The suite includes the standard `testing/slogtest` contract checks, JSON/Text
group parity, duplicate-key checks against raw output, record isolation, error
and panic propagation, and 128 concurrent request contexts in both modes.

Benchmarks use `LogAttrs` without per-call attributes, JSONHandler/TextHandler
writing to `io.Discard`, and one context lookup plus an integer conversion per
dynamic field. Construction and `With` binding are outside the timed loop.
They measure logging and formatting, not disk/network I/O. The grouped cases
use two groups and two bound ordinary attributes. Fixed fields are preformatted
by native slog; dynamic fields must be recomputed and formatted per event.

Measured on 2026-10-14 with Go 1.26.0, darwin/arm64, Apple M4, `GOMAXPROCS=1`.
The table reports the median of five runs; bytes and allocations were identical
across those runs and both output formats.

```sh
GOTOOLCHAIN=go1.26.0 go test -run '^$' -bench BenchmarkHandler -benchmem -count=5 -cpu=1
```

| Case | JSON ns/op | Text ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Native, no fields | 258.0 | 263.7 | 0 | 0 |
| Native `With`, 3 fixed fields | 262.1 | 259.8 | 0 | 0 |
| 1 dynamic field | 337.4 | 332.9 | 0 | 0 |
| 3 dynamic fields | 413.2 | 411.7 | 0 | 0 |
| 10 dynamic fields | 874.5 | 903.6 | 208 | 1 |
| Filtered, native | 3.332 | 3.369 | 0 | 0 |
| Filtered, 10 registrations | 3.980 | 3.919 | 0 | 0 |
| Two groups, 3 fixed fields | 267.5 | 259.8 | 0 | 0 |
| Two groups, 3 dynamic fields at root | 749.8 | 655.7 | 128 | 2 |
| Two groups, 3 current-group dynamic fields | 440.8 | 464.2 | 0 | 0 |

For JSON, three dynamic fields cost approximately **151 ns/op** more than three
preformatted fixed fields in this workload. Default two-level grouping adds
approximately **309 ns/op and two allocations** compared with current-group
delegation. Filtered logs perform no context extraction and add less than one
nanosecond in these measurements. Small differences between native fixed and
no-field cases are measurement variation, not evidence of a speedup.

The small ungrouped path uses stack scratch storage and batches
`Record.AddAttrs` to avoid repeated overflow growth. Ten fields exceed slog's
inline attribute storage; nonempty group trees need per-event storage in default
mode. Records with more attributes, deep groups, larger values, expensive
callbacks, or a different toolchain can have different allocation and timing
costs. No pooling, reflection, goroutine-local state, or asynchronous mechanism
is introduced.
