# Context-aware slog enrichment

`enrich` adds context-aware attributes to a standard `slog.Handler`. Use the
ordinary `slog.Logger`, including `InfoContext`, `LogAttrs`, `With`, and
`WithGroup`. The module requires Go 1.26.0. The implementation uses only the
standard library; tests use `github.com/stretchr/testify` for assertions.

## Installation

```sh
go get github.com/go-fries/fries/log/slog/enrich/v4
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

	"github.com/go-fries/fries/log/slog/enrich/v4"
)

type userIDKey struct{}

func userID(ctx context.Context) slog.Value {
	id, ok := ctx.Value(userIDKey{}).(int64)
	if !ok {
		return enrich.Omit()
	}
	return slog.Int64Value(id)
}

func main() {
	logger := slog.New(enrich.NewHandler(
		slog.NewJSONHandler(os.Stdout, nil),
		enrich.WithValue("user_id", userID),
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

### Extract batches of attributes

Use an `Extractor` when one context lookup can provide several attributes.
`WithExtractors` accepts one or more callbacks. This setup fragment assumes
`base` is a handler and the application stores request metadata as shown:

```go
type requestMetadata struct {
	UserID    int64
	RequestID string
}

type metadataKey struct{}

func requestAttrs(ctx context.Context) []slog.Attr {
	metadata, ok := ctx.Value(metadataKey{}).(requestMetadata)
	if !ok {
		return nil
	}
	return []slog.Attr{
		slog.Int64("user_id", metadata.UserID),
		slog.String("request_id", metadata.RequestID),
	}
}

func cancellationAttrs(ctx context.Context) []slog.Attr {
	return []slog.Attr{slog.Bool("canceled", ctx.Err() != nil)}
}

// Inside application setup:
handler := enrich.NewHandler(base,
	enrich.WithExtractors(requestAttrs, cancellationAttrs),
)
logger := slog.New(handler)
```

`WithValue` and `WithExtractors` may be mixed. For example, a single value,
two extractors, and another single value execute in exactly that sequence.
Each extractor's returned attributes stay together and retain their slice order.
Returning `nil` or an empty slice adds no attributes. Passing no extractors is a
no-op; nil extractors are ignored without affecting the remaining callbacks.

`WithExtractors` copies its argument slice when the option is created, so it can
be reused without depending on later changes to that slice. The handler copies
returned attribute entries into per-record storage and never modifies the
returned slice. It does not deep-copy nested groups or `Any` values. Returned
data must remain immutable while it is being handled; shared slices and values
must be read-only. Do not reuse a global writable result buffer across requests.

## API and field behavior

```go
type Valuer func(context.Context) slog.Value
type Extractor func(context.Context) []slog.Attr

func NewHandler(next slog.Handler, opts ...Option) *Handler
func WithValue(key string, value Valuer) Option
func WithExtractors(extractors ...Extractor) Option
func WithCurrentGroup() Option
func Omit() slog.Value
```

`WithValue` follows the repository's `WithXxx` option naming convention. Values
use standard `slog.StringValue`, `Int64Value`, `AnyValue`, etc.; there are no
parallel conversion helpers or custom logger APIs.

| Situation | Behavior |
| --- | --- |
| Evaluation | Each Valuer and Extractor runs once per `Handle`, in mixed registration order, before the child handler runs. `Enabled`, `WithAttrs`, and `WithGroup` execute neither kind of callback. |
| Filtered event | `slog.Logger` does not call `Handle`; no callback executes. Direct `Handle` calls assume the caller has checked `Enabled`, as in native slog. |
| `0`, `false`, `""` | Retained. |
| Missing field | Return `Omit()`, an empty `slog.GroupValue()`, for a single value; return `nil` or an empty slice for an extractor. |
| Zero `slog.Value` | A nil value, normally rendered as JSON `null`; it is not an omission sentinel. An empty key with a nil value is the zero Attr and follows slog's omission rule. |
| Group value | Passed to the child handler; nested attributes are retained. An empty group is omitted; an empty group key inlines its contents. |
| Duplicate keys | Retained, including repeated registrations, bound fields, record fields, and collisions with built-in keys. No overwrite or deduplication guarantee. |
| Field order | The child's existing and root-bound fields precede registered dynamic fields. Dynamic fields precede record fields. Within a group, bound fields precede record fields. The child can transform or reorder output. |
| Nil context | Standard Logger methods substitute `context.Background()`. Use Logger methods rather than calling Handler methods with nil contexts. |
| Canceled context | Still passed through and evaluated; cancellation does not suppress logging. |
| Callback panic | Propagates to the caller; this module does not recover it. Native `slog.LogValuer` resolution remains the child handler's responsibility. |
| Handler error | Returned unchanged by `Handle`. Standard `slog.Logger` discards handler errors; use a child error-reporting handler if required. |
| Nil constructor inputs | A nil handler panics during construction; nil Valuer/Extractor registrations are ignored. |

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
handler := enrich.NewHandler(
	slog.NewJSONHandler(os.Stdout, nil),
	enrich.WithCurrentGroup(),
	enrich.WithValue("user_id", userID),
)
logger := slog.New(handler).WithGroup("order").With("source", "marketplace")
logger.InfoContext(ctx, "created", "order_id", 1001)
```

Here the output contains:

```json
{"order":{"source":"marketplace","user_id":123,"order_id":1001}}
```

Both `WithAttrs` and `WithGroup` delegate to the child. Dynamic attributes are still
computed during `Handle`, never during binding. This option applies to all
registered Valuers and Extractors; it does not mix global and group-local registrations.

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
- This component is a registration-based injector. Passing a Valuer or Extractor
  directly as an ordinary `logger.With` or log-call value does not register it;
  use `WithValue` or `WithExtractors`.
- It does not merge duplicate fields or promise compatibility with arbitrary
  third-party grouping policies. JSON key order is not an overwrite policy.

## Validation and performance

From the repository root:

```sh
make test/log/slog/enrich
make test/log/slog/enrich ARGS='-race -count=1'
make lint/log/slog/enrich
```

From this module directory:

```sh
go test ./...
go test -race ./...
go vet ./...
go test -run '^$' -bench BenchmarkHandler -benchmem -count=3
```

The suite includes the standard `testing/slogtest` contract checks, JSON/Text
group parity, duplicate-key checks against raw output, record isolation, error
and panic propagation, mixed registration order, option/result slice ownership,
and 128 concurrent request contexts using both callback types in both modes.

Benchmarks use `LogAttrs` without per-call attributes, JSONHandler/TextHandler
writing to `io.Discard`. Valuers perform one context lookup per field; extractors
perform one lookup per batch and construct a fresh attribute slice. A separate
read-only extractor returns a prebuilt immutable slice from the context.
Construction, prebuilding that slice, and `With` binding are outside the timed loop.
They measure logging and formatting, not disk/network I/O. The grouped cases
use two groups and two bound ordinary attributes. Fixed fields are preformatted
by native slog; dynamic fields must be recomputed and formatted per event.

Measured on 2026-10-15 with Go 1.26.0, darwin/arm64, Apple M4, `GOMAXPROCS=1`.
The table reports the median of three runs; bytes and allocations were identical
across those runs and both output formats.

```sh
GOTOOLCHAIN=go1.26.0 go test -run '^$' -bench BenchmarkHandler -benchmem -count=3 -cpu=1
```

| Case | JSON ns/op | Text ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: | ---: |
| Native, no fields | 261.9 | 256.9 | 0 | 0 |
| Native `With`, 3 fixed fields | 263.5 | 260.7 | 0 | 0 |
| 1 Valuer | 347.3 | 329.5 | 0 | 0 |
| 3 Valuers | 414.5 | 412.7 | 0 | 0 |
| 10 Valuers | 895.5 | 879.8 | 208 | 1 |
| Extractor, 1 field in a fresh slice | 403.8 | 377.0 | 48 | 1 |
| Extractor, 3 fields in a fresh slice | 528.5 | 511.9 | 128 | 1 |
| Extractor, 10 fields in a fresh slice | 978.4 | 947.2 | 624 | 2 |
| Extractor, 3 prebuilt read-only fields | 427.6 | 406.6 | 0 | 0 |
| Filtered, native | 3.473 | 3.457 | 0 | 0 |
| Filtered, 10 Valuers | 4.034 | 3.985 | 0 | 0 |
| Filtered, 1 Extractor | 3.858 | 3.933 | 0 | 0 |
| Two groups, 3 fixed fields | 265.3 | 263.2 | 0 | 0 |
| Two groups, 3 Valuers at root | 727.0 | 728.5 | 128 | 2 |
| Two groups, 3 current-group Valuers | 439.1 | 468.3 | 0 | 0 |
| Two groups, Extractor with 3 fields at root | 749.6 | 760.3 | 256 | 3 |
| Two groups, current-group Extractor with 3 fields | 527.3 | 572.9 | 128 | 1 |

For JSON, three dynamic fields cost approximately **151 ns/op** more than three
preformatted fixed fields in this workload. Default two-level grouping adds
approximately **288 ns/op and two allocations** compared with current-group
delegation for three Valuers. Filtered logs perform no context extraction and
add less than one nanosecond in these measurements. Small differences between
native fixed and no-field cases are measurement variation, not evidence of a speedup.

Batch extraction is not automatically faster: constructing a fresh slice for
three fields costs about **114 ns/op and one allocation** more than three Valuers
in the JSON workload, despite performing only one context lookup. Returning
immutable attributes already stored in the context avoids that allocation. The
single-value path does not adapt Valuers into one-element slices. Do not use a
shared writable buffer to reduce extractor allocations; that can mix request
data or cause races.

The small ungrouped path uses stack scratch storage and batches
`Record.AddAttrs` to avoid repeated overflow growth. Ten fields exceed slog's
inline attribute storage; nonempty group trees need per-event storage in default
mode. Records with more attributes, deep groups, larger values, expensive
callbacks, or a different toolchain can have different allocation and timing
costs. No pooling, reflection, goroutine-local state, or asynchronous mechanism
is introduced.
