package gin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func middleware1(t *testing.T) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("middleware1", "middleware1")
		assert.Equal(t, "/ping", c.Request.URL.Path)
		c.Next()
	}
}

func middleware2(t *testing.T) gin.HandlerFunc {
	return func(c *gin.Context) {
		assert.Equal(t, "middleware1", c.MustGet("middleware1").(string))
		c.Set("middleware2", "middleware2")
		assert.Equal(t, "/ping", c.Request.URL.Path)
		c.Next()
	}
}

func TestServer(t *testing.T) {
	srv := NewServer(
		gin.New(),
		Addr(":8080"),
		Middleware(
			middleware1(t),
			middleware2(t),
		),
	)

	srv.GET("/ping", func(c *gin.Context) {
		assert.Equal(t, http.MethodGet, c.Request.Method)
		assert.Equal(t, "/ping", c.Request.URL.Path)
		assert.Equal(t, "middleware1", c.MustGet("middleware1").(string))
		assert.Equal(t, "middleware2", c.MustGet("middleware2").(string))
		c.String(http.StatusOK, "pong")
	})

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
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
}
