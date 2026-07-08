package server

import (
	"context"
	"errors"
	"net/http"
)

// IgnoreServerClosed returns a server that treats [http.ErrServerClosed] from Start as nil.
func IgnoreServerClosed(srv interface {
	Start(context.Context) error
	Stop(context.Context) error
},
) IgnoreClosed {
	return IgnoreClosed{srv: srv}
}

// IgnoreClosed wraps a server and ignores [http.ErrServerClosed] returned by Start.
type IgnoreClosed struct {
	srv server
}

// Start starts the wrapped server and returns nil for [http.ErrServerClosed].
func (s IgnoreClosed) Start(ctx context.Context) error {
	err := s.srv.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop stops the wrapped server.
func (s IgnoreClosed) Stop(ctx context.Context) error {
	return s.srv.Stop(ctx)
}
