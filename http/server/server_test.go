package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	ctx := context.Background()
	srv := NewWithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "server", r.URL.Query().Get("name"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}), WithHTTPServerAddr(":8081"))

	go func(ctx context.Context) {
		_ = srv.Start(ctx)
	}(ctx)
	defer func(ctx context.Context) {
		assert.NoError(t, srv.Stop(ctx))
	}(ctx)

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Error("failed to start server")
		default:
			resp, err := http.Get("http://localhost:8081?name=server")
			if err == nil && resp.StatusCode == http.StatusOK {
				_ = resp.Body.Close()
				t.Log("server started")
				return
			}
		}
	}
}
