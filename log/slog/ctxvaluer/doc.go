// Package ctxvaluer adds context-aware attributes to standard slog handlers.
//
// By default, dynamic attributes remain outside groups created on the wrapper.
// The wrapper represents those groups with slog.GroupValue instead of calling
// the wrapped handler's WithGroup. Construct the wrapper around an ungrouped
// handler to keep dynamic attributes at the output's top level.
//
// Root WithAttrs calls delegate. Grouped WithAttrs are retained until Handle,
// so ordinary grouped slog.LogValuers and ReplaceAttr callbacks may run on every
// event. Default mode cannot preserve child policies triggered specifically by
// WithGroup or grouped WithAttrs; the child must support slog.GroupValue.
//
// WithCurrentGroup selects native delegation: all WithAttrs and WithGroup calls
// reach the wrapped handler, and dynamic attributes follow its current group.
//
// Values are evaluated once per handled record, in registration order. Duplicate
// keys are preserved, not merged. Valuer panics and handler errors propagate;
// slog.Logger itself discards handler errors, as specified by slog.
package ctxvaluer
