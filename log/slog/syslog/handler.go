//go:build !windows && !plan9

package syslog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

var _ slog.Handler = (*Handler)(nil)

// Writer writes syslog messages at a selected priority.
type Writer interface {
	// Debug writes a debug-priority syslog message.
	Debug(string) error
	// Info writes an info-priority syslog message.
	Info(string) error
	// Warning writes a warning-priority syslog message.
	Warning(string) error
	// Err writes an error-priority syslog message.
	Err(string) error
}

type attrEntry struct {
	groups []string
	attr   slog.Attr
}

// Handler writes slog records to syslog.
type Handler struct {
	writer Writer
	level  slog.Leveler
	attrs  []attrEntry
	groups []string
}

// NewHandler creates a [Handler] that writes records to writer.
func NewHandler(writer Writer, opts ...Option) *Handler {
	return newHandler(writer, newConfig(opts...))
}

func newHandler(writer Writer, cfg *config) *Handler {
	return &Handler{
		writer: writer,
		level:  cfg.level,
	}
}

// Enabled reports whether level is enabled by the handler configuration.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	if h.writer == nil {
		return false
	}
	if h.level == nil {
		return true
	}
	return level >= h.level.Level()
}

// Handle writes record to syslog.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	if !h.Enabled(ctx, record.Level) {
		return nil
	}

	message := h.formatRecord(record)
	switch {
	case record.Level < slog.LevelInfo:
		return h.writer.Debug(message)
	case record.Level < slog.LevelWarn:
		return h.writer.Info(message)
	case record.Level < slog.LevelError:
		return h.writer.Warning(message)
	default:
		return h.writer.Err(message)
	}
}

// WithAttrs returns a new handler whose records include attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	clone := h.clone()
	for _, attr := range attrs {
		clone.attrs = append(clone.attrs, attrEntry{
			groups: slices.Clone(h.groups),
			attr:   attr,
		})
	}
	return clone
}

// WithGroup returns a new handler with name appended to the current group.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	clone := h.clone()
	clone.groups = append(clone.groups, name)
	return clone
}

// Close closes the underlying writer when it implements Close.
func (h *Handler) Close() error {
	closer, ok := h.writer.(interface{ Close() error })
	if !ok {
		return nil
	}
	return closer.Close()
}

func (h *Handler) clone() *Handler {
	clone := &Handler{
		writer: h.writer,
		level:  h.level,
		attrs:  slices.Clone(h.attrs),
		groups: slices.Clone(h.groups),
	}
	return clone
}

func (h *Handler) formatRecord(record slog.Record) string {
	var buf bytes.Buffer
	buf.WriteString(record.Message)
	for _, entry := range h.attrs {
		appendAttr(&buf, entry.groups, entry.attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		appendAttr(&buf, h.groups, attr)
		return true
	})
	return buf.String()
}

func appendAttr(buf *bytes.Buffer, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}

	if attr.Value.Kind() == slog.KindGroup {
		attrs := attr.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if attr.Key != "" {
			groups = append(slices.Clone(groups), attr.Key)
		}
		for _, groupAttr := range attrs {
			appendAttr(buf, groups, groupAttr)
		}
		return
	}

	if attr.Key == "" {
		return
	}

	buf.WriteByte(' ')
	buf.WriteString(qualifiedKey(groups, attr.Key))
	buf.WriteByte('=')
	buf.WriteString(fmt.Sprint(attr.Value.Any()))
}

func qualifiedKey(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}
	return strings.Join(append(slices.Clone(groups), key), ".")
}
