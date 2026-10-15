package ctxvaluer

import (
	"context"
	"log/slog"
)

// Valuer computes an attribute value from the context of the current log call.
//
// A Valuer must be safe for concurrent use. Return the value produced by [Omit]
// to omit the attribute. A zero [slog.Value] is a nil value, not an omission;
// numeric zero, false, and the empty string are retained.
//
// Panics propagate to the caller of [Handler.Handle].
type Valuer func(context.Context) slog.Value

// Omit returns an empty group value, equivalent to calling [slog.GroupValue]
// with no attributes. Handlers omit the value regardless of the attribute's key.
func Omit() slog.Value {
	return slog.GroupValue()
}
