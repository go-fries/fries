package ctxvaluer

import (
	"context"
	"log/slog"
	"slices"
)

var _ slog.Handler = (*Handler)(nil)

type group struct {
	name  string
	attrs []slog.Attr
}

// Handler injects context-aware attributes before a record's attributes.
// Configuration and derived handlers are immutable. Concurrent use is safe if
// the wrapped handler and each Valuer are safe for concurrent use.
type Handler struct {
	next   slog.Handler
	config *config
	groups []group
}

// NewHandler wraps next. It panics if next or a registered Valuer is nil.
// By default, dynamic attributes remain outside groups created on this wrapper;
// next must be ungrouped if output-level top-level attributes are required.
// Use WithCurrentGroup for fully delegated group behavior.
func NewHandler(next slog.Handler, opts ...Option) *Handler {
	if next == nil {
		panic("ctxvaluer: nil handler")
	}
	return &Handler{next: next, config: newConfig(opts...)}
}

// Enabled delegates to the wrapped handler without evaluating any Valuer.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle evaluates registered values once, in order, and passes a new record to
// the wrapped handler. It preserves record metadata and does not modify shared
// record attributes. Errors and Valuer panics propagate unchanged.
// Like other slog handlers, Handle assumes Enabled has already returned true.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if len(h.config.fields) == 0 && len(h.groups) == 0 {
		return h.next.Handle(ctx, record)
	}

	// Small, ungrouped records need no scratch allocation. AddAttrs receives all
	// attributes together so it can grow the record's overflow storage only once.
	var buf [16]slog.Attr
	attrs := buf[:0]
	size := len(h.config.fields) + record.NumAttrs()
	if len(h.groups) > 0 {
		size = len(h.config.fields) + 1
	}
	if size > len(buf) {
		attrs = make([]slog.Attr, 0, size)
	}
	for _, f := range h.config.fields {
		value := f.value(ctx)
		if value.Kind() == slog.KindGroup && len(value.Group()) == 0 {
			continue
		}
		attrs = append(attrs, slog.Attr{Key: f.key, Value: value})
	}
	if len(h.groups) == 0 {
		record.Attrs(func(a slog.Attr) bool {
			attrs = append(attrs, a)
			return true
		})
	} else if a := h.groupAttr(record); a.Value.Kind() == slog.KindGroup {
		attrs = append(attrs, a)
	}
	out := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	out.AddAttrs(attrs...)
	return h.next.Handle(ctx, out)
}

// groupAttr builds a single tree, not separate group attributes for WithAttrs
// and record attrs. Separate objects would produce duplicate JSON group keys.
func (h *Handler) groupAttr(record slog.Record) slog.Attr {
	var child slog.Attr
	for i, g := range slices.Backward(h.groups) {
		size := len(g.attrs)
		if i == len(h.groups)-1 {
			size += record.NumAttrs()
		} else if child.Value.Kind() == slog.KindGroup {
			size++
		}
		if size == 0 {
			continue
		}
		attrs := make([]slog.Attr, 0, size)
		attrs = append(attrs, g.attrs...)
		if i == len(h.groups)-1 {
			record.Attrs(func(a slog.Attr) bool {
				attrs = append(attrs, a)
				return true
			})
		} else if child.Value.Kind() == slog.KindGroup {
			attrs = append(attrs, child)
		}
		child = slog.Attr{Key: g.name, Value: slog.GroupValue(attrs...)}
	}
	return child
}

// WithAttrs returns a derived handler. Root attributes delegate to next. In the
// default mode, grouped attributes are retained and sent as GroupValue contents
// during Handle; their ordinary slog.LogValuers may therefore resolve per event.
// Context Valuers are never evaluated here.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	clone := *h
	if len(h.groups) == 0 {
		clone.next = h.next.WithAttrs(attrs)
	} else {
		clone.groups = slices.Clone(h.groups)
		last := &clone.groups[len(clone.groups)-1]
		last.attrs = slices.Concat(last.attrs, attrs)
	}
	return &clone
}

// WithGroup returns a derived handler. An empty name returns h. By default the
// wrapper owns the group, keeping dynamic attributes outside it. WithCurrentGroup
// instead delegates to next, so dynamic attributes inherit the native group.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	if h.config.currentGroup {
		clone.next = h.next.WithGroup(name)
	} else {
		clone.groups = append(slices.Clone(h.groups), group{name: name})
	}
	return &clone
}
