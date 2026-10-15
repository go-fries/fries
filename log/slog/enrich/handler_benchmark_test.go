package enrich_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/go-fries/fries/log/slog/enrich/v4"
	"github.com/stretchr/testify/require"
)

type benchmarkKey struct{}

type benchmarkData struct {
	values [10]int64
	attrs  [10]slog.Attr
}

func benchmarkOptions(count int) []enrich.Option {
	opts := make([]enrich.Option, 0, count)
	for i := range count {
		opts = append(opts, enrich.WithValue(fmt.Sprintf("field_%d", i), func(ctx context.Context) slog.Value {
			data := ctx.Value(benchmarkKey{}).(*benchmarkData)
			return slog.Int64Value(data.values[i])
		}))
	}
	return opts
}

func benchmarkExtractor(count int) enrich.Extractor {
	keys := make([]string, count)
	for i := range keys {
		keys[i] = fmt.Sprintf("field_%d", i)
	}
	return func(ctx context.Context) []slog.Attr {
		data := ctx.Value(benchmarkKey{}).(*benchmarkData)
		attrs := make([]slog.Attr, count)
		for i, key := range keys {
			attrs[i] = slog.Int64(key, data.values[i])
		}
		return attrs
	}
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
				logger := slog.New(enrich.NewHandler(factory(), benchmarkOptions(count)...))
				benchmarkLog(b, logger, slog.LevelInfo)
			})
			b.Run(fmt.Sprintf("%s/extractor_%d", format, count), func(b *testing.B) {
				logger := slog.New(enrich.NewHandler(factory(), enrich.WithExtractors(benchmarkExtractor(count))))
				benchmarkLog(b, logger, slog.LevelInfo)
			})
		}
		b.Run(format+"/extractor_readonly_3", func(b *testing.B) {
			logger := slog.New(enrich.NewHandler(factory(), enrich.WithExtractors(func(ctx context.Context) []slog.Attr {
				return ctx.Value(benchmarkKey{}).(*benchmarkData).attrs[:3]
			})))
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		b.Run(format+"/filtered_native", func(b *testing.B) {
			benchmarkLog(b, slog.New(factory()), slog.LevelDebug)
		})
		b.Run(format+"/filtered_dynamic_10", func(b *testing.B) {
			opts := benchmarkOptions(10)
			opts[0] = enrich.WithValue("never", func(context.Context) slog.Value {
				require.FailNow(b, "filtered log evaluated a Valuer")
				return enrich.Omit()
			})
			logger := slog.New(enrich.NewHandler(factory(), opts...))
			benchmarkLog(b, logger, slog.LevelDebug)
		})
		b.Run(format+"/filtered_extractor", func(b *testing.B) {
			logger := slog.New(enrich.NewHandler(factory(), enrich.WithExtractors(func(context.Context) []slog.Attr {
				require.FailNow(b, "filtered log evaluated an Extractor")
				return nil
			})))
			benchmarkLog(b, logger, slog.LevelDebug)
		})
		b.Run(format+"/grouped_native_fixed_3", func(b *testing.B) {
			logger := slog.New(factory()).WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card").
				With("field_0", int64(0), "field_1", int64(1), "field_2", int64(2))
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		b.Run(format+"/grouped_dynamic_3", func(b *testing.B) {
			logger := slog.New(enrich.NewHandler(factory(), benchmarkOptions(3)...)).
				WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card")
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		b.Run(format+"/grouped_current_dynamic_3", func(b *testing.B) {
			opts := append(benchmarkOptions(3), enrich.WithCurrentGroup())
			logger := slog.New(enrich.NewHandler(factory(), opts...)).
				WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card")
			benchmarkLog(b, logger, slog.LevelInfo)
		})
		for _, currentGroup := range []bool{false, true} {
			name := "grouped_extractor_3"
			if currentGroup {
				name = "grouped_current_extractor_3"
			}
			b.Run(format+"/"+name, func(b *testing.B) {
				opts := []enrich.Option{enrich.WithExtractors(benchmarkExtractor(3))}
				if currentGroup {
					opts = append(opts, enrich.WithCurrentGroup())
				}
				logger := slog.New(enrich.NewHandler(factory(), opts...)).
					WithGroup("order").With("source", "http").WithGroup("payment").With("method", "card")
				benchmarkLog(b, logger, slog.LevelInfo)
			})
		}
	}
}

func benchmarkLog(b *testing.B, logger *slog.Logger, level slog.Level) {
	data := &benchmarkData{}
	for i := range data.values {
		data.values[i] = int64(i)
		data.attrs[i] = slog.Int64(fmt.Sprintf("field_%d", i), int64(i))
	}
	ctx := context.WithValue(b.Context(), benchmarkKey{}, data)
	b.ReportAllocs()
	for b.Loop() {
		logger.LogAttrs(ctx, level, "created")
	}
}
