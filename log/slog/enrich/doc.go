// Package enrich adds context-aware attributes to [log/slog] handlers.
//
// Register individual values with [WithValue] or batches of attributes with
// [WithExtractors], then wrap a [slog.Handler] with [NewHandler]. The resulting
// [Handler] works with the standard [slog.Logger]. Use methods such as
// [slog.Logger.InfoContext] to supply the context for each log call.
//
// # Groups
//
// By default, dynamic attributes remain outside groups created on the wrapper.
// The wrapper constructs those groups with [slog.GroupValue] instead of delegating
// to [slog.Handler.WithGroup]. Construct the wrapper around an ungrouped handler
// to keep dynamic attributes at the output's top level.
//
// [Handler.WithAttrs] delegates root attributes and retains grouped attributes
// until [Handler.Handle]. Grouped [slog.LogValuer] values and
// [slog.HandlerOptions.ReplaceAttr] callbacks may therefore run for each record.
// Default mode requires a wrapped handler that supports group values. It cannot
// preserve policies triggered specifically by the wrapped handler's
// [slog.Handler.WithGroup] or grouped [slog.Handler.WithAttrs] calls.
//
// [WithCurrentGroup] selects native delegation: [Handler.WithAttrs] and
// [Handler.WithGroup] delegate to the wrapped handler, and dynamic attributes
// follow its current group.
//
// # Evaluation and errors
//
// [Handler.Handle] evaluates each registered [Valuer] and [Extractor] once per
// handled record, in mixed registration order. Each extractor's attribute order
// is preserved. Duplicate keys are preserved, not merged. Return the value
// produced by [Omit] to omit a single value, or a nil or empty slice to omit a
// batch. A zero [slog.Value] is a nil value, not an omission.
//
// Panics from registered callbacks and errors from the wrapped handler propagate
// unchanged. The standard [slog.Logger] discards handler errors.
// Nil [Valuer] and [Extractor] registrations are ignored; [NewHandler] panics
// if the wrapped handler is nil.
package enrich
