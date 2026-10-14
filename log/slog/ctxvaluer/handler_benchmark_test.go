package ctxvaluer_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
)

type benchmarkKey struct{}

func benchmarkOptions(count int) []ctxvaluer.Option {
	opts := make([]ctxvaluer.Option, 0, count)
	for i := range count {
		opts = append(opts, ctxvaluer.WithValue(fmt.Sprintf("field_%d", i), func(ctx context.Context) slog.Value {
			values := ctx.Value(benchmarkKey{}).(*[10]int64)
			return slog.Int64Value(values[i])
		}))
	}
	return opts
}

func BenchmarkHandler(b *testing.B) {
	for _, format := range []string{"json", "text"} {
		factory := func() slog.Handler {
			if format == "json" {
				return slog.NewJSONHandler(io.Discard, nil)
			}
			return slog.NewTextHandler(io.Discard, nil)
		}
		b.Run(format+"/native", func(b *testing.B) {
			benchmarkLog(b, slog.New(factory()), slog.LevelInfo)
		})
		b.Run(format+"/fixed_3", func(b *testing.B) {
			logger := slog.New(factory()).With("field_0", int64(0), "field_1", int64(1), "field_2", int64(2))
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		for _, count := range []int{1, 3, 10} {
			b.Run(fmt.Sprintf("%s/dynamic_%d", format, count), func(b *testing.B) {
				logger := slog.New(ctxvaluer.NewHandler(factory(), benchmarkOptions(count)...))
				benchmarkLog(b, logger, slog.LevelInfo)
			})
		}
		b.Run(format+"/filtered_native", func(b *testing.B) {
			benchmarkLog(b, slog.New(factory()), slog.LevelDebug)
		})
		b.Run(format+"/filtered_dynamic_10", func(b *testing.B) {
			opts := benchmarkOptions(10)
			opts[0] = ctxvaluer.WithValue("never", func(context.Context) slog.Value {
				panic("filtered log evaluated a Valuer")
			})
			logger := slog.New(ctxvaluer.NewHandler(factory(), opts...))
			benchmarkLog(b, logger, slog.LevelDebug)
		})
		b.Run(format+"/grouped_native_fixed_3", func(b *testing.B) {
			logger := slog.New(factory()).WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card").
				With("field_0", int64(0), "field_1", int64(1), "field_2", int64(2))
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		b.Run(format+"/grouped_dynamic_3", func(b *testing.B) {
			logger := slog.New(ctxvaluer.NewHandler(factory(), benchmarkOptions(3)...)).
				WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card")
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		b.Run(format+"/grouped_current_dynamic_3", func(b *testing.B) {
			opts := append(benchmarkOptions(3), ctxvaluer.WithCurrentGroup())
			logger := slog.New(ctxvaluer.NewHandler(factory(), opts...)).
				WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card")
			benchmarkLog(b, logger, slog.LevelInfo)
		})
	}
}

func benchmarkLog(b *testing.B, logger *slog.Logger, level slog.Level) {
	values := &[10]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	ctx := context.WithValue(b.Context(), benchmarkKey{}, values)
	b.ReportAllocs()
	for b.Loop() {
		logger.LogAttrs(ctx, level, "created")
	}
}
