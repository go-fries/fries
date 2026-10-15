package enrich_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/go-fries/fries/log/slog/enrich/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractorsMixedRegistrationOrder(t *testing.T) {
	ctx := context.WithValue(t.Context(), contextKey{}, 123)
	var order []string
	var buf bytes.Buffer
	logger := slog.New(enrich.NewHandler(slog.NewJSONHandler(&buf, outputOptions()),
		enrich.WithValue("duplicate", func(context.Context) slog.Value {
			order = append(order, "value:first")
			return slog.IntValue(0)
		}),
		enrich.WithExtractors(
			func(got context.Context) []slog.Attr {
				assert.Same(t, ctx, got)
				order = append(order, "extract:first")
				return []slog.Attr{slog.Int("duplicate", 1), slog.Int("duplicate", 2)}
			},
			func(context.Context) []slog.Attr { order = append(order, "extract:nil"); return nil },
			func(context.Context) []slog.Attr { order = append(order, "extract:empty"); return []slog.Attr{} },
			func(context.Context) []slog.Attr {
				order = append(order, "extract:last")
				return []slog.Attr{slog.Int("duplicate", 3)}
			},
		),
		enrich.WithValue("duplicate", func(context.Context) slog.Value {
			order = append(order, "value:last")
			return slog.IntValue(4)
		}),
	)).With("duplicate", -1)
	_ = logger.WithGroup("unused").With("bound", 1)
	assert.True(t, logger.Enabled(ctx, slog.LevelInfo))
	assert.Empty(t, order, "binding and Enabled must not evaluate callbacks")
	logger.InfoContext(ctx, "mixed", "duplicate", 5)
	logger.LogAttrs(ctx, slog.LevelInfo, "mixed", slog.Int("duplicate", 5))
	wantOrder := []string{"value:first", "extract:first", "extract:nil", "extract:empty", "extract:last", "value:last"}
	assert.Equal(t, append(slices.Clone(wantOrder), wantOrder...), order)
	// Keep raw duplicate keys and their order; JSONEq would discard that evidence.
	want := "{\"level\":\"INFO\",\"msg\":\"mixed\",\"duplicate\":-1,\"duplicate\":0,\"duplicate\":1,\"duplicate\":2,\"duplicate\":3,\"duplicate\":4,\"duplicate\":5}\n"
	assert.Equal(t, want+want, buf.String())
}

func TestExtractorsFilteringAndContexts(t *testing.T) {
	var buf bytes.Buffer
	var calls int
	logger := slog.New(enrich.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), enrich.WithExtractors(
		func(ctx context.Context) []slog.Attr {
			require.NotNil(t, ctx)
			calls++
			id, ok := ctx.Value(contextKey{}).(int)
			if !ok {
				return nil
			}
			return []slog.Attr{slog.Int("user_id", id)}
		},
	)))
	logger.DebugContext(t.Context(), "filtered")
	assert.Zero(t, calls)
	logger.InfoContext(nil, "nil") //nolint:staticcheck // Exercise slog's documented nil-context normalization.
	ctx, cancel := context.WithCancel(context.WithValue(t.Context(), contextKey{}, 123))
	cancel()
	logger.InfoContext(ctx, "canceled")
	logger.InfoContext(context.WithValue(t.Context(), contextKey{}, 456), "another")
	assert.Equal(t, 3, calls)
	results := decodeLines(t, buf.Bytes())
	require.Len(t, results, 3)
	assert.NotContains(t, results[0], "user_id")
	assert.Equal(t, float64(123), results[1]["user_id"])
	assert.Equal(t, float64(456), results[2]["user_id"])
}

func TestExtractorValuesAndGroupScope(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		for _, currentGroup := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/current_group=%t", format, currentGroup), func(t *testing.T) {
				var buf bytes.Buffer
				var next slog.Handler = slog.NewJSONHandler(&buf, outputOptions())
				if format == "text" {
					next = slog.NewTextHandler(&buf, outputOptions())
				}
				opts := []enrich.Option{enrich.WithExtractors(func(context.Context) []slog.Attr {
					return []slog.Attr{
						slog.Int("zero", 0), slog.Bool("false", false), slog.String("empty", ""), slog.Any("nil", nil),
						slog.Group("nested", slog.String("key", "value")), slog.Group("", slog.String("inline", "ok")),
						{},
						slog.Group("ignored"),
						{Key: "omitted", Value: enrich.Omit()},
					}
				})}
				if currentGroup {
					opts = append(opts, enrich.WithCurrentGroup())
				}
				logger := slog.New(enrich.NewHandler(next, opts...)).WithGroup("order").With("source", "http").
					WithGroup("payment").With("method", "card")
				logger.InfoContext(t.Context(), "values", "id", 1)
				want := "{\"level\":\"INFO\",\"msg\":\"values\",\"zero\":0,\"false\":false,\"empty\":\"\",\"nil\":null,\"nested\":{\"key\":\"value\"},\"inline\":\"ok\",\"order\":{\"source\":\"http\",\"payment\":{\"method\":\"card\",\"id\":1}}}\n"
				if currentGroup {
					want = "{\"level\":\"INFO\",\"msg\":\"values\",\"order\":{\"source\":\"http\",\"payment\":{\"method\":\"card\",\"zero\":0,\"false\":false,\"empty\":\"\",\"nil\":null,\"nested\":{\"key\":\"value\"},\"inline\":\"ok\",\"id\":1}}}\n"
				}
				if format == "text" {
					want = "level=INFO msg=values zero=0 false=false empty=\"\" nil=<nil> nested.key=value inline=ok order.source=http order.payment.method=card order.payment.id=1\n"
					if currentGroup {
						want = "level=INFO msg=values order.source=http order.payment.method=card order.payment.zero=0 order.payment.false=false order.payment.empty=\"\" order.payment.nil=<nil> order.payment.nested.key=value order.payment.inline=ok order.payment.id=1\n"
					}
				}
				assert.Equal(t, want, buf.String())
			})
		}
	}
}

func TestExtractorsOptionSnapshotAndReuse(t *testing.T) {
	extractors := []enrich.Extractor{
		func(context.Context) []slog.Attr { return []slog.Attr{slog.Int("first", 1)} },
		func(context.Context) []slog.Attr { return []slog.Attr{slog.Int("second", 2)} },
	}
	opt := enrich.WithExtractors(extractors...)
	extractors[0] = nil
	for range 2 {
		var buf bytes.Buffer
		logger := slog.New(enrich.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opt))
		logger.InfoContext(t.Context(), "snapshot")
		assert.Equal(t, "{\"level\":\"INFO\",\"msg\":\"snapshot\",\"first\":1,\"second\":2}\n", buf.String())
	}
}

func TestExtractorsValidationAndEmptyOption(t *testing.T) {
	valid := func(context.Context) []slog.Attr {
		require.FailNow(t, "constructor must not invoke extractors")
		return nil
	}
	for _, callbacks := range [][]enrich.Extractor{{nil}, {valid, nil}, {nil, valid}} {
		opt := enrich.WithExtractors(callbacks...)
		assert.PanicsWithValue(t, "enrich: nil extractor", func() {
			enrich.NewHandler(slog.NewJSONHandler(io.Discard, nil), opt)
		})
	}
	var buf bytes.Buffer
	slog.New(enrich.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), enrich.WithExtractors())).InfoContext(t.Context(), "empty")
	assert.Equal(t, "{\"level\":\"INFO\",\"msg\":\"empty\"}\n", buf.String())
}

func TestExtractorLargeResultAndRecordIsolation(t *testing.T) {
	attrs := make([]slog.Attr, 24)
	for i := range attrs {
		attrs[i] = slog.Int(fmt.Sprintf("batch_%d", i), i)
	}
	before := slices.Clone(attrs)
	record := slog.NewRecord(time.Unix(123, 456), slog.LevelWarn, "original", 12345)
	for i := range 9 {
		record.AddAttrs(slog.Int(fmt.Sprintf("record_%d", i), i))
	}
	original := recordAttrs(record)
	want := slices.Concat([]slog.Attr{slog.Int("first", 1)}, attrs, []slog.Attr{slog.Int("last", 2)}, original)
	var calls int
	h := enrich.NewHandler(&probeHandler{handle: func(ctx context.Context, got slog.Record) error {
		calls++
		assert.Same(t, t.Context(), ctx)
		assert.Equal(t, record.Time, got.Time)
		assert.Equal(t, record.Level, got.Level)
		assert.Equal(t, record.Message, got.Message)
		assert.Equal(t, record.PC, got.PC)
		assert.Equal(t, want, recordAttrs(got))
		got.AddAttrs(slog.Int("child_mutation", 1))
		return nil
	}}, enrich.WithValue("first", func(context.Context) slog.Value { return slog.IntValue(1) }),
		enrich.WithExtractors(func(context.Context) []slog.Attr { return attrs }),
		enrich.WithValue("last", func(context.Context) slog.Value { return slog.IntValue(2) }))
	for range 2 {
		require.NoError(t, h.Handle(t.Context(), record))
	}
	assert.Equal(t, 2, calls)
	assert.Equal(t, before, attrs, "extractor results must not be modified")
	assert.Equal(t, original, recordAttrs(record), "original record must not be modified")
}

func TestExtractorErrorAndPanicPropagation(t *testing.T) {
	want := errors.New("child failed")
	h := enrich.NewHandler(&probeHandler{handle: func(context.Context, slog.Record) error { return want }},
		enrich.WithExtractors(func(context.Context) []slog.Attr { return nil }))
	err := h.Handle(t.Context(), slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0))
	require.ErrorIs(t, err, want)
	assert.Same(t, want, err)
	var handled, laterCalled bool
	h = enrich.NewHandler(&probeHandler{handle: func(context.Context, slog.Record) error { handled = true; return nil }},
		enrich.WithExtractors(func(context.Context) []slog.Attr { panic(want) }, func(context.Context) []slog.Attr {
			laterCalled = true
			return nil
		}))
	assert.PanicsWithValue(t, want, func() { slog.New(h).InfoContext(t.Context(), "panic") })
	assert.False(t, handled)
	assert.False(t, laterCalled)
}

func TestExtractorSlogHandlerConformance(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		t.Run(fmt.Sprintf("current_group=%t", currentGroup), func(t *testing.T) {
			var buf bytes.Buffer
			opts := []enrich.Option{enrich.WithExtractors(func(context.Context) []slog.Attr {
				if currentGroup {
					// Preserve the suite's expectation that empty groups stay empty.
					return nil
				}
				return []slog.Attr{slog.Int("extracted", 1)}
			})}
			if currentGroup {
				opts = append(opts, enrich.WithCurrentGroup())
			}
			h := enrich.NewHandler(slog.NewJSONHandler(&buf, nil), opts...)
			require.NoError(t, slogtest.TestHandler(h, func() []map[string]any { return decodeLines(t, buf.Bytes()) }))
		})
	}
}
