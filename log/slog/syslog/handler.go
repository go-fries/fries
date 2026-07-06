//go:build !windows && !plan9

package syslog

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"sync"
	"time"
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
	mu     *sync.Mutex
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
		mu:     &sync.Mutex{},
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
	h.mu.Lock()
	defer h.mu.Unlock()

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

	h.mu.Lock()
	defer h.mu.Unlock()
	return closer.Close()
}

func (h *Handler) clone() *Handler {
	clone := &Handler{
		writer: h.writer,
		level:  h.level,
		mu:     h.mu,
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
	writeQualifiedKey(buf, groups, attr.Key)
	buf.WriteByte('=')
	appendValue(buf, attr.Value)
}

func writeQualifiedKey(buf *bytes.Buffer, groups []string, key string) {
	for _, group := range groups {
		buf.WriteString(group)
		buf.WriteByte('.')
	}
	buf.WriteString(key)
}

func appendValue(buf *bytes.Buffer, value slog.Value) {
	switch value.Kind() {
	case slog.KindString:
		s := value.String()
		if needsQuoting(s) {
			buf.WriteString(strconv.Quote(s))
		} else {
			buf.WriteString(s)
		}
	case slog.KindInt64:
		buf.WriteString(strconv.FormatInt(value.Int64(), 10))
	case slog.KindUint64:
		buf.WriteString(strconv.FormatUint(value.Uint64(), 10))
	case slog.KindFloat64:
		buf.WriteString(strconv.FormatFloat(value.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.WriteString(strconv.FormatBool(value.Bool()))
	case slog.KindDuration:
		buf.WriteString(value.Duration().String())
	case slog.KindTime:
		buf.WriteString(value.Time().Format(time.RFC3339))
	default:
		buf.WriteString(fmt.Sprint(value.Any()))
	}
}

func needsQuoting(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c == '=' || c == '"' {
			return true
		}
	}
	return false
}
