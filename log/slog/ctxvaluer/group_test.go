package ctxvaluer_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
)

func TestGroupParityWithNativeHandlers(t *testing.T) {
	factories := map[string]func(io.Writer, *slog.HandlerOptions) slog.Handler{
		"json": func(w io.Writer, o *slog.HandlerOptions) slog.Handler { return slog.NewJSONHandler(w, o) },
		"text": func(w io.Writer, o *slog.HandlerOptions) slog.Handler { return slog.NewTextHandler(w, o) },
	}
	cases := []struct {
		name  string
		bind  func(*slog.Logger) *slog.Logger
		attrs []slog.Attr
	}{
		{"single", func(l *slog.Logger) *slog.Logger { return l.WithGroup("order") }, []slog.Attr{slog.Int("id", 1)}},
		{"multiple", func(l *slog.Logger) *slog.Logger { return l.WithGroup("order").WithGroup("payment") }, []slog.Attr{slog.Int("id", 1)}},
		{"bound_then_record", func(l *slog.Logger) *slog.Logger { return l.WithGroup("order").With("source", "marketplace") }, []slog.Attr{slog.Int("id", 1)}},
		{"mixed", func(l *slog.Logger) *slog.Logger {
			return l.With("service", "api").WithGroup("order").With("source", "marketplace").
				WithGroup("payment").With("method", "card").With("currency", "USD")
		}, []slog.Attr{slog.Int("id", 1)}},
		{"empty_record", func(l *slog.Logger) *slog.Logger {
			return l.WithGroup("order").With("source", "marketplace").WithGroup("empty")
		}, nil},
		{"nested_empty_record", func(l *slog.Logger) *slog.Logger {
			return l.WithGroup("order").With("source", "marketplace").WithGroup("empty").WithGroup("nested")
		}, nil},
		{"all_empty", func(l *slog.Logger) *slog.Logger { return l.WithGroup("empty").WithGroup("nested") }, nil},
		{"zero_attr", func(l *slog.Logger) *slog.Logger { return l.WithGroup("empty").With(slog.Attr{}) }, []slog.Attr{{}}},
		{"empty_group", func(l *slog.Logger) *slog.Logger { return l.WithGroup("empty").With(slog.Group("nested")) }, []slog.Attr{slog.Group("ignored")}},
		{"inline_group", func(l *slog.Logger) *slog.Logger {
			return l.WithGroup("order").With(slog.Group("", slog.Int("bound", 1)))
		}, []slog.Attr{slog.Group("", slog.Int("inline", 2))}},
		{"empty_name", func(l *slog.Logger) *slog.Logger { return l.WithGroup("").WithGroup("order").WithGroup("") }, []slog.Attr{slog.Int("id", 1)}},
	}
	for format, factory := range factories {
		for _, tc := range cases {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				var native, wrapped bytes.Buffer
				baseline := tc.bind(slog.New(factory(&native, outputOptions())))
				baseline.LogAttrs(t.Context(), slog.LevelInfo, "message", tc.attrs...)
				logger := tc.bind(slog.New(ctxvaluer.NewHandler(factory(&wrapped, outputOptions()),
					ctxvaluer.WithValue("missing", func(context.Context) slog.Value { return ctxvaluer.Omit() }),
				)))
				logger.LogAttrs(t.Context(), slog.LevelInfo, "message", tc.attrs...)
				if wrapped.String() != native.String() {
					t.Fatalf("wrapped:\n%s\nnative:\n%s", wrapped.String(), native.String())
				}
			})
		}
	}
}

func TestDynamicGroupScope(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var buf bytes.Buffer
			opts := []ctxvaluer.Option{ctxvaluer.WithValue("user_id", contextID)}
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...)).
				WithGroup("order").With("source", "marketplace").WithGroup("payment").With("method", "card")
			logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 123), "created", "id", 1001)
			want := "{\"level\":\"INFO\",\"msg\":\"created\",\"user_id\":123,\"order\":{\"source\":\"marketplace\",\"payment\":{\"method\":\"card\",\"id\":1001}}}\n"
			if currentGroup {
				want = "{\"level\":\"INFO\",\"msg\":\"created\",\"order\":{\"source\":\"marketplace\",\"payment\":{\"method\":\"card\",\"user_id\":123,\"id\":1001}}}\n"
			}
			if got := buf.String(); got != want {
				t.Fatalf("output:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestGroupedDuplicatesAreNotMerged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), ctxvaluer.WithValue("id", contextID))).
		WithGroup("order").With("id", 1).With("id", 2)
	logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 0), "duplicates", "id", 3)
	want := "{\"level\":\"INFO\",\"msg\":\"duplicates\",\"id\":0,\"order\":{\"id\":1,\"id\":2,\"id\":3}}\n"
	if got := buf.String(); got != want {
		t.Fatalf("output=%q, want %q", got, want)
	}
}

func TestEmptyRecordGroupScope(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var buf bytes.Buffer
			opts := []ctxvaluer.Option{ctxvaluer.WithValue("user_id", contextID)}
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...)).WithGroup("order")
			logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 123), "empty")
			want := "{\"level\":\"INFO\",\"msg\":\"empty\",\"user_id\":123}\n"
			if currentGroup {
				want = "{\"level\":\"INFO\",\"msg\":\"empty\",\"order\":{\"user_id\":123}}\n"
			}
			if got := buf.String(); got != want {
				t.Fatalf("output=%q, want %q", got, want)
			}
		})
	}
}

func TestEmptyGroupsDoNotAddRecordAttributes(t *testing.T) {
	opts := make([]ctxvaluer.Option, 0, 5)
	for i := range 5 {
		opts = append(opts, ctxvaluer.WithValue(fmt.Sprintf("dynamic_%d", i), func(context.Context) slog.Value {
			return slog.IntValue(i)
		}))
	}
	next := &probeHandler{handle: func(_ context.Context, record slog.Record) error {
		if record.NumAttrs() != 5 {
			t.Errorf("empty groups added attributes: %v", recordAttrs(record))
		}
		return nil
	}}
	slog.New(ctxvaluer.NewHandler(next, opts...)).WithGroup("empty").WithGroup("nested").InfoContext(t.Context(), "message")
}

func TestTextDynamicFieldsRemainAtRoot(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(ctxvaluer.NewHandler(slog.NewTextHandler(&buf, outputOptions()), ctxvaluer.WithValue("user_id", contextID))).
		WithGroup("order").With("source", "marketplace")
	logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 123), "created", "id", 1001)
	want := "level=INFO msg=created user_id=123 order.source=marketplace order.id=1001\n"
	if got := buf.String(); got != want {
		t.Fatalf("output=%q, want %q", got, want)
	}
}

func TestDerivedHandlersAreIndependent(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), ctxvaluer.WithValue("user_id", contextID)))
	parent := base.WithGroup("order").With("shared", 1)
	left := parent.With("side", "left").WithGroup("payment").With("method", "card")
	right := parent.With("side", "right").WithGroup("shipment").With("carrier", "mail")
	ctx := context.WithValue(t.Context(), contextKey{}, 123)
	base.InfoContext(ctx, "base")
	parent.InfoContext(ctx, "parent")
	left.InfoContext(ctx, "left", "id", 2)
	right.InfoContext(ctx, "right", "id", 3)
	want := "{\"level\":\"INFO\",\"msg\":\"base\",\"user_id\":123}\n" +
		"{\"level\":\"INFO\",\"msg\":\"parent\",\"user_id\":123,\"order\":{\"shared\":1}}\n" +
		"{\"level\":\"INFO\",\"msg\":\"left\",\"user_id\":123,\"order\":{\"shared\":1,\"side\":\"left\",\"payment\":{\"method\":\"card\",\"id\":2}}}\n" +
		"{\"level\":\"INFO\",\"msg\":\"right\",\"user_id\":123,\"order\":{\"shared\":1,\"side\":\"right\",\"shipment\":{\"carrier\":\"mail\",\"id\":3}}}\n"
	if got := buf.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestDelegationModes(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var calls []string
			next := &probeHandler{}
			next.withAttrs = func(attrs []slog.Attr) slog.Handler {
				calls = append(calls, "attrs:"+attrs[0].Key)
				return next
			}
			next.withGroup = func(name string) slog.Handler {
				calls = append(calls, "group:"+name)
				return next
			}
			var opts []ctxvaluer.Option
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			slog.New(ctxvaluer.NewHandler(next, opts...)).With("root", 1).WithGroup("order").With("bound", 2)
			want := []string{"attrs:root"}
			if currentGroup {
				want = append(want, "group:order", "attrs:bound")
			}
			if !slices.Equal(calls, want) {
				t.Fatalf("delegation=%v, want %v", calls, want)
			}
		})
	}
}

func TestReplaceAttrReceivesCorrectGroupPaths(t *testing.T) {
	var paths []string
	next := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "dynamic" || a.Key == "bound" || a.Key == "record" {
			paths = append(paths, fmt.Sprintf("%s:%v", a.Key, groups))
		}
		return a
	}})
	logger := slog.New(ctxvaluer.NewHandler(next, ctxvaluer.WithValue("dynamic", func(context.Context) slog.Value { return slog.IntValue(1) }))).
		WithGroup("order").With("bound", 2).WithGroup("payment")
	logger.InfoContext(t.Context(), "message", "record", 3)
	want := []string{"dynamic:[]", "bound:[order]", "record:[order payment]"}
	if !slices.Equal(paths, want) {
		t.Fatalf("paths=%v, want %v", paths, want)
	}
}

type logValueFunc func() slog.Value

func (f logValueFunc) LogValue() slog.Value { return f() }

func TestLogValuerResolutionTiming(t *testing.T) {
	var rootCalls, groupCalls, dynamicCalls int
	logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(io.Discard, nil),
		ctxvaluer.WithValue("dynamic", func(context.Context) slog.Value {
			return slog.AnyValue(logValueFunc(func() slog.Value {
				dynamicCalls++
				return slog.IntValue(1)
			}))
		}),
	)).With("root", logValueFunc(func() slog.Value { rootCalls++; return slog.IntValue(1) })).
		WithGroup("group").With("bound", logValueFunc(func() slog.Value { groupCalls++; return slog.IntValue(1) }))
	if rootCalls != 1 || groupCalls != 0 || dynamicCalls != 0 {
		t.Fatalf("construction calls: root=%d group=%d dynamic=%d", rootCalls, groupCalls, dynamicCalls)
	}
	for range 2 {
		logger.InfoContext(t.Context(), "message")
	}
	if rootCalls != 1 || groupCalls != 2 || dynamicCalls != 2 {
		t.Fatalf("logging calls: root=%d group=%d dynamic=%d", rootCalls, groupCalls, dynamicCalls)
	}
}

func TestPreGroupedHandlerCannotBeUngrouped(t *testing.T) {
	var buf bytes.Buffer
	next := slog.NewJSONHandler(&buf, outputOptions()).WithGroup("existing")
	logger := slog.New(ctxvaluer.NewHandler(next, ctxvaluer.WithValue("user_id", contextID))).WithGroup("order")
	logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 123), "message", "id", 1)
	want := "{\"level\":\"INFO\",\"msg\":\"message\",\"existing\":{\"user_id\":123,\"order\":{\"id\":1}}}\n"
	if got := buf.String(); got != want {
		t.Fatalf("output=%q, want %q", got, want)
	}
}
