package ctxvaluer_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerConcurrentSafe(t *testing.T) {
	for _, currentGroup := range []bool{false, true} {
		name := "root"
		if currentGroup {
			name = "current_group"
		}
		t.Run(name, func(t *testing.T) {
			const workers, logs = 128, 4
			var buf bytes.Buffer
			var calls atomic.Int64
			opts := []ctxvaluer.Option{ctxvaluer.WithValue("user_id", func(ctx context.Context) slog.Value {
				calls.Add(1)
				return contextID(ctx)
			})}
			if currentGroup {
				opts = append(opts, ctxvaluer.WithCurrentGroup())
			}
			shared := slog.New(ctxvaluer.NewHandler(slog.NewJSONHandler(&buf, outputOptions()), opts...)).
				With("service", "api").WithGroup("request").With("source", "http")
			var wg sync.WaitGroup
			for id := range workers {
				wg.Go(func() {
					ctx := context.WithValue(t.Context(), contextKey{}, id)
					for n := range logs {
						if n%2 == 0 {
							shared.InfoContext(ctx, "shared", "expected", id, "sequence", n)
						} else {
							// Derive while other goroutines use the shared parent.
							shared.With("bound", id).WithGroup("job").LogAttrs(ctx, slog.LevelInfo, "derived",
								slog.Int("expected", id), slog.Int("sequence", n))
						}
					}
				})
			}
			wg.Wait()
			results := decodeLines(t, buf.Bytes())
			assert.Len(t, results, workers*logs)
			assert.Equal(t, int64(workers*logs), calls.Load())
			seen := make(map[[2]int]bool, workers*logs)
			for _, result := range results {
				request, ok := result["request"].(map[string]any)
				require.True(t, ok, "missing request group: %v", result)
				fields := request
				if result["msg"] == "derived" {
					fields, ok = request["job"].(map[string]any)
					require.True(t, ok, "missing job group: %v", result)
					assert.Equal(t, fields["expected"], request["bound"], "derived fields crossed contexts: %v", result)
				}
				id := result["user_id"]
				if currentGroup {
					id = fields["user_id"]
				}
				require.IsType(t, float64(0), id)
				require.IsType(t, float64(0), fields["sequence"])
				assert.Equal(t, fields["expected"], id, "fields crossed contexts: %v", result)
				assert.Equal(t, "api", result["service"])
				assert.Equal(t, "http", request["source"])
				key := [2]int{int(id.(float64)), int(fields["sequence"].(float64))}
				assert.NotContains(t, seen, key, "duplicate record: %v", key)
				seen[key] = true
			}
		})
	}
}
