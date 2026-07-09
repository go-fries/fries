package server_test

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-fries/fries/http/server/v4"
)

func Example_directUse() {
	srv := server.New(&http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello world"))
		}),
	})

	if err := srv.Start(context.Background()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}
