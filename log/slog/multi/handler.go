package multi

import (
	"context"
	"errors"
	"log/slog"
)

var _ slog.Handler = (*Handler)(nil)

// Handler dispatches slog records to multiple child handlers.
type Handler struct {
	handlers []slog.Handler
}

// NewHandler creates a [Handler] that dispatches records to each supplied
// handler. Nil handlers are ignored.
func NewHandler(handlers ...slog.Handler) *Handler {
	h := &Handler{}
	for _, handler := range handlers {
		if handler != nil {
			h.handlers = append(h.handlers, handler)
		}
	}
	return h
}

// Enabled reports whether any child handler is enabled for the provided level.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle dispatches the record to every enabled child handler. Errors returned
// by child handlers are combined with [errors.Join].
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range h.handlers {
		if !handler.Enabled(ctx, record.Level) {
			continue
		}
		if err := handler.Handle(ctx, record.Clone()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithAttrs returns a new handler whose child handlers include the provided
// attributes.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := &Handler{
		handlers: make([]slog.Handler, 0, len(h.handlers)),
	}
	for _, handler := range h.handlers {
		clone.handlers = append(clone.handlers, handler.WithAttrs(attrs))
	}
	return clone
}

// WithGroup returns a new handler whose child handlers include the provided
// group.
func (h *Handler) WithGroup(name string) slog.Handler {
	clone := &Handler{
		handlers: make([]slog.Handler, 0, len(h.handlers)),
	}
	for _, handler := range h.handlers {
		clone.handlers = append(clone.handlers, handler.WithGroup(name))
	}
	return clone
}
