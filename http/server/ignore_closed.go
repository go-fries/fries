package server

import (
	"context"
	"errors"
	"net/http"
)

// IgnoreServerClosed returns a server that treats [http.ErrServerClosed] from Start as nil.
func IgnoreServerClosed(srv server) server {
	return ignoreClosed{srv: srv}
}

type ignoreClosed struct {
	srv server
}

func (s ignoreClosed) Start(ctx context.Context) error {
	err := s.srv.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s ignoreClosed) Stop(ctx context.Context) error {
	return s.srv.Stop(ctx)
}
