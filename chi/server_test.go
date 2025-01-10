package chi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	srv := NewServer(chi.NewRouter(), Addr(":8001"))

	srv.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte("pong"))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		assert.ErrorIs(t, srv.Start(ctx), http.ErrServerClosed)
	}()

	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequest(http.MethodGet, "http://"+srv.addr+"/ping", nil)
	assert.NoError(t, err)

	recorder := httptest.NewRecorder()
	srv.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "pong", recorder.Body.String())

	assert.NoError(t, srv.Stop(ctx))
}
