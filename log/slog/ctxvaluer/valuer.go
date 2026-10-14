package ctxvaluer

import (
	"context"
	"log/slog"
)

// Valuer computes an attribute from the context of the current log call.
// It must be safe for concurrent use. Return Omit() to omit the attribute.
// A zero slog.Value is a nil value, not an omission; 0, false and "" are retained.
// Panics propagate to the caller.
type Valuer func(context.Context) slog.Value

// Omit returns an empty group value, which slog handlers omit regardless of key.
func Omit() slog.Value {
	return slog.GroupValue()
}
