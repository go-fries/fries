package enrich_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/go-fries/fries/log/slog/enrich/v4"
)

type requestIDKey struct{}

func requestID(ctx context.Context) slog.Value {
	id, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return enrich.Omit()
	}
	return slog.StringValue(id)
}

func ExampleNewHandler() {
	logger := slog.New(enrich.NewHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}), enrich.WithValue("request_id", requestID))).WithGroup("order").With("source", "marketplace")

	// Set request metadata once at the HTTP boundary. Business code only passes
	// the request context to the standard slog.Logger.
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), "created", "order_id", 1001)
		w.WriteHeader(http.StatusCreated)
	})
	middleware := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if id := r.Header.Get("X-Request-ID"); id != "" {
			ctx = context.WithValue(ctx, requestIDKey{}, id)
		}
		endpoint.ServeHTTP(w, r.WithContext(ctx))
	})
	request := httptest.NewRequest(http.MethodPost, "/orders", nil)
	request.Header.Set("X-Request-ID", "abc123")
	middleware.ServeHTTP(httptest.NewRecorder(), request)
	// Output:
	// {"level":"INFO","msg":"created","request_id":"abc123","order":{"source":"marketplace","order_id":1001}}
}

func ExampleWithCurrentGroup() {
	logger := slog.New(enrich.NewHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}), enrich.WithCurrentGroup(), enrich.WithValue("request_id", requestID))).WithGroup("order")
	ctx := context.WithValue(context.Background(), requestIDKey{}, "abc123")
	logger.InfoContext(ctx, "created", "order_id", 1001)
	// Output:
	// level=INFO msg=created order.request_id=abc123 order.order_id=1001
}

func ExampleWithExtractors() {
	logger := slog.New(enrich.NewHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}), enrich.WithExtractors(
		func(ctx context.Context) []slog.Attr {
			id, ok := ctx.Value(requestIDKey{}).(string)
			if !ok {
				return nil
			}
			return []slog.Attr{slog.String("request_id", id), slog.String("source", "http")}
		},
		func(ctx context.Context) []slog.Attr {
			return []slog.Attr{slog.Bool("canceled", ctx.Err() != nil)}
		},
	))).WithGroup("order")
	ctx := context.WithValue(context.Background(), requestIDKey{}, "abc123")
	logger.InfoContext(ctx, "created", "order_id", 1001)
	// Output:
	// level=INFO msg=created request_id=abc123 source=http canceled=false order.order_id=1001
}
