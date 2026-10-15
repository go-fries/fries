package enrich

import (
	"context"
	"log/slog"
)

// Extractor extracts log attributes from the context of the current log call.
// A nil or empty result adds no attributes.
//
// An Extractor must be safe for concurrent use and must not mutate its result
// while it is being handled. Shared slices and values must be immutable.
// [Handler] copies the outer attribute entries, but does not deep-copy group
// contents or values stored by [slog.AnyValue].
//
// Panics propagate to the caller of [Handler.Handle].
type Extractor func(context.Context) []slog.Attr
