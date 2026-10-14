package ctxvaluer_test

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
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
			if len(results) != workers*logs || calls.Load() != workers*logs {
				t.Fatalf("records=%d calls=%d, want %d", len(results), calls.Load(), workers*logs)
			}
			seen := make(map[[2]int]bool, workers*logs)
			for _, result := range results {
				request := result["request"].(map[string]any)
				fields := request
				if result["msg"] == "derived" {
					fields = request["job"].(map[string]any)
					if request["bound"] != fields["expected"] {
						t.Fatalf("derived fields crossed contexts: %v", result)
					}
				}
				id := result["user_id"]
				if currentGroup {
					id = fields["user_id"]
				}
				if id != fields["expected"] || result["service"] != "api" || request["source"] != "http" {
					t.Fatalf("fields crossed contexts: %v", result)
				}
				key := [2]int{int(id.(float64)), int(fields["sequence"].(float64))}
				if seen[key] {
					t.Fatalf("duplicate record: %v", key)
				}
				seen[key] = true
			}
		})
	}
}
