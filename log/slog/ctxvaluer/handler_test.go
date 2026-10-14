package ctxvaluer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
)

type contextKey struct{}

func contextID(ctx context.Context) slog.Value {
	id, ok := ctx.Value(contextKey{}).(int)
	if !ok {
		return ctxvaluer.Omit()
	}
	return slog.IntValue(id)
}

func outputOptions() *slog.HandlerOptions {
	return &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}
}

func TestInfoContextAndLogAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()),
		ctxvaluer.WithValue("user_id", contextID),
	))
	logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 123), "created", "order_id", 1001)
	logger.LogAttrs(context.WithValue(t.Context(), contextKey{}, 456), slog.LevelInfo, "paid", slog.Int("order_id", 1002))
	want := "{\"level\":\"INFO\",\"msg\":\"created\",\"user_id\":123,\"order_id\":1001}\n" +
		"{\"level\":\"INFO\",\"msg\":\"paid\",\"user_id\":456,\"order_id\":1002}\n"
	if got := buf.String(); got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestRegistrationOrderAndExactlyOnce(t *testing.T) {
	var order []int
	opts := make([]ctxvaluer.Option, 0, 3)
	for i := range 3 {
		opts = append(opts, ctxvaluer.WithValue(fmt.Sprintf("value_%d", i), func(context.Context) slog.Value {
			order = append(order, i)
			return slog.IntValue(i)
		}))
	}
	var buf bytes.Buffer
	logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...)).
		With("service", "api").WithGroup("order").With("source", "marketplace")
	if len(order) != 0 {
		t.Fatal("With/WithGroup evaluated values")
	}
	logger.InfoContext(t.Context(), "first", "id", 1)
	logger.LogAttrs(t.Context(), slog.LevelInfo, "second", slog.Int("id", 2))
	if !slices.Equal(order, []int{0, 1, 2, 0, 1, 2}) {
		t.Fatalf("evaluation order: %v", order)
	}
	if !stringsContainInOrder(buf.String(), `"value_0":0`, `"value_1":1`, `"value_2":2`, `"order":`) {
		t.Fatalf("attribute order: %s", buf.String())
	}
}

func stringsContainInOrder(s string, parts ...string) bool {
	for _, part := range parts {
		i := strings.Index(s, part)
		if i < 0 {
			return false
		}
		s = s[i+len(part):]
	}
	return true
}

func TestEnabledDelegatesWithoutEvaluation(t *testing.T) {
	ctx := context.WithValue(t.Context(), contextKey{}, 42)
	var levels []slog.Level
	var calls int
	next := &probeHandler{enabled: func(got context.Context, level slog.Level) bool {
		if got != ctx {
			t.Error("Enabled received a different context")
		}
		levels = append(levels, level)
		return level >= slog.LevelWarn
	}}
	h := ctxvaluer.NewHandler(next, ctxvaluer.WithValue("id", func(context.Context) slog.Value {
		calls++
		return slog.IntValue(1)
	}))
	logger := slog.New(h)
	logger.DebugContext(ctx, "filtered")
	logger.InfoContext(ctx, "filtered")
	if !logger.Enabled(ctx, slog.LevelWarn) {
		t.Error("enabled level not delegated")
	}
	if calls != 0 || !slices.Equal(levels, []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn}) {
		t.Fatalf("values=%d, levels=%v", calls, levels)
	}
}

func TestValuesAndDuplicateKeys(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var buf bytes.Buffer
			opts := []ctxvaluer.Option{
				ctxvaluer.WithValue("zero", func(context.Context) slog.Value { return slog.Int64Value(0) }),
				ctxvaluer.WithValue("false", func(context.Context) slog.Value { return slog.BoolValue(false) }),
				ctxvaluer.WithValue("empty", func(context.Context) slog.Value { return slog.StringValue("") }),
				ctxvaluer.WithValue("nil", func(context.Context) slog.Value { return slog.Value{} }),
				ctxvaluer.WithValue("missing", func(context.Context) slog.Value { return ctxvaluer.Omit() }),
				ctxvaluer.WithValue("empty_group", func(context.Context) slog.Value { return slog.GroupValue() }),
				ctxvaluer.WithValue("nested", func(context.Context) slog.Value {
					return slog.GroupValue(slog.String("name", "value"), slog.Group("inner", slog.Int("id", 7)))
				}),
				ctxvaluer.WithValue("", func(context.Context) slog.Value {
					return slog.GroupValue(slog.String("inline", "value"))
				}),
				ctxvaluer.WithValue("duplicate", func(context.Context) slog.Value { return slog.IntValue(1) }),
				ctxvaluer.WithValue("duplicate", func(context.Context) slog.Value { return slog.IntValue(2) }),
			}
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...)).
				With("duplicate", 0)
			logger.InfoContext(t.Context(), "values", "duplicate", 3, slog.Attr{}, slog.Group("ignored"))
			// Check raw output, not a map: decoding duplicate keys would discard data.
			want := "{\"level\":\"INFO\",\"msg\":\"values\",\"duplicate\":0,\"zero\":0,\"false\":false,\"empty\":\"\",\"nil\":null," +
				"\"nested\":{\"name\":\"value\",\"inner\":{\"id\":7}},\"inline\":\"value\",\"duplicate\":1,\"duplicate\":2,\"duplicate\":3}\n"
			if got := buf.String(); got != want {
				t.Fatalf("output:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

func TestNilAndCanceledContext(t *testing.T) {
	var buf bytes.Buffer
	var calls int
	logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()),
		ctxvaluer.WithValue("id", func(ctx context.Context) slog.Value {
			calls++
			if ctx == nil {
				t.Fatal("slog passed a nil context")
			}
			return contextID(ctx)
		}),
	))
	logger.InfoContext(nil, "nil") //nolint:staticcheck // Exercise slog's documented nil-context normalization.
	logger.Info("background")
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, 42))
	cancel()
	logger.InfoContext(ctx, "canceled")
	if calls != 3 {
		t.Fatalf("calls=%d, want 3", calls)
	}
	results := decodeLines(t, buf.Bytes())
	if _, ok := results[0]["id"]; ok {
		t.Fatal("missing context value was not omitted")
	}
	if got := results[2]["id"]; got != float64(42) {
		t.Fatalf("canceled context lost its value: %v", got)
	}
}

func TestRecordIsolationAndMetadata(t *testing.T) {
	for _, grouped := range []bool{false, true} {
		t.Run(fmt.Sprintf("grouped=%t", grouped), func(t *testing.T) {
			record := slog.NewRecord(time.Unix(123, 456), slog.LevelWarn, "message", 12345)
			for i := range 9 {
				record.AddAttrs(slog.Int(fmt.Sprintf("key_%d", i), i))
			}
			before := recordAttrs(record)
			var count int
			next := &probeHandler{handle: func(ctx context.Context, got slog.Record) error {
				count++
				if ctx != t.Context() || !got.Time.Equal(record.Time) || got.Level != record.Level ||
					got.Message != record.Message || got.PC != record.PC {
					t.Error("context or record metadata changed")
				}
				attrs := recordAttrs(got)
				if attrs[0].Key != "dynamic" {
					t.Errorf("unexpected attributes: %v", attrs)
				}
				if grouped {
					attrs = attrs[1].Value.Group()
				} else {
					attrs = attrs[1:]
				}
				if !reflect.DeepEqual(attrs, before) {
					t.Errorf("record attrs changed: %v", attrs)
				}
				got.AddAttrs(slog.String("child", "mutation"))
				return nil
			}}
			var h slog.Handler = ctxvaluer.NewHandler(next, ctxvaluer.WithValue("dynamic", func(context.Context) slog.Value {
				return slog.IntValue(1)
			}))
			if grouped {
				h = h.WithGroup("group")
			}
			for range 2 {
				if err := h.Handle(t.Context(), record); err != nil {
					t.Fatal(err)
				}
			}
			if count != 2 || !reflect.DeepEqual(recordAttrs(record), before) {
				t.Fatal("handling modified the original record")
			}
			record.AddAttrs(slog.Int("after", 10))
			if record.NumAttrs() != 10 {
				t.Fatalf("unsafe record copy detected: %v", recordAttrs(record))
			}
		})
	}
}

func TestLargeRegistrationAndOptionReuse(t *testing.T) {
	opts := make([]ctxvaluer.Option, 0, 20)
	for i := range 20 {
		opts = append(opts, ctxvaluer.WithValue(fmt.Sprintf("dynamic_%d", i), func(context.Context) slog.Value {
			return slog.IntValue(i)
		}))
	}
	for range 2 {
		var buf bytes.Buffer
		logger := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...))
		logger.InfoContext(t.Context(), "large", "record", 21)
		result := decodeLines(t, buf.Bytes())[0]
		if len(result) != 23 || result["dynamic_19"] != float64(19) || result["record"] != float64(21) {
			t.Fatalf("large registration lost fields: %v", result)
		}
	}
}

func TestHandleErrorAndPanicPropagation(t *testing.T) {
	want := errors.New("write failed")
	h := ctxvaluer.NewHandler(&probeHandler{handle: func(context.Context, slog.Record) error { return want }},
		ctxvaluer.WithValue("id", contextID))
	if got := h.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0)); got != want {
		t.Fatalf("error=%v, want %v", got, want)
	}
	var handled bool
	h = ctxvaluer.NewHandler(&probeHandler{handle: func(context.Context, slog.Record) error {
		handled = true
		return nil
	}}, ctxvaluer.WithValue("panic", func(context.Context) slog.Value { panic(want) }))
	defer func() {
		if got := recover(); got != want {
			t.Errorf("panic=%v, want %v", got, want)
		}
		if handled {
			t.Error("child handled a record after a Valuer panic")
		}
	}()
	slog.New(h).InfoContext(t.Context(), "panic")
}

func TestConstructorValidationAndEmptyDerivation(t *testing.T) {
	for name, create := range map[string]func(){
		"nil_handler": func() { ctxvaluer.NewHandler(nil) },
		"nil_valuer":  func() { ctxvaluer.NewHandler(slog.NewJSONHandler(io.Discard, nil), ctxvaluer.WithValue("id", nil)) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("constructor did not panic")
				}
			}()
			create()
		})
	}
	h := ctxvaluer.NewHandler(slog.NewJSONHandler(io.Discard, nil))
	if h.WithAttrs(nil) != h || h.WithGroup("") != h {
		t.Error("empty derivation did not return the receiver")
	}
	if err := h.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "empty", 0)); err != nil {
		t.Fatal(err)
	}
}

func TestSlogHandlerConformance(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var buf bytes.Buffer
			value := slog.IntValue(1)
			// The conformance suite expects empty record groups to stay empty.
			// A present current-group value would intentionally make them nonempty.
			if currentGroup {
				value = ctxvaluer.Omit()
			}
			opts := []ctxvaluer.Option{ctxvaluer.WithValue("dynamic", func(context.Context) slog.Value { return value })}
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			h := ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, nil), opts...)
			if err := slogtest.TestHandler(h, func() []map[string]any { return decodeLines(t, buf.Bytes()) }); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func decodeLines(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(data), []byte{'\n'})
	results := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var result map[string]any
		if err := json.Unmarshal(line, &result); err != nil {
			t.Fatalf("invalid JSON %s: %v", line, err)
		}
		results = append(results, result)
	}
	return results
}

func recordAttrs(record slog.Record) []slog.Attr {
	attrs := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, a)
		return true
	})
	return attrs
}

type probeHandler struct {
	enabled   func(context.Context, slog.Level) bool
	handle    func(context.Context, slog.Record) error
	withAttrs func([]slog.Attr) slog.Handler
	withGroup func(string) slog.Handler
}

func (h *probeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.enabled == nil || h.enabled(ctx, level)
}

func (h *probeHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.handle != nil {
		return h.handle(ctx, record)
	}
	return nil
}

func (h *probeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.withAttrs != nil {
		return h.withAttrs(attrs)
	}
	return h
}

func (h *probeHandler) WithGroup(name string) slog.Handler {
	if h.withGroup != nil {
		return h.withGroup(name)
	}
	return h
}
