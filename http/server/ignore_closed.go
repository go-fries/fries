package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-fries/fries/contract/v4"
)

// IgnoreServerClosed returns a server that treats [http.ErrServerClosed] from Start as nil.
func IgnoreServerClosed(srv contract.Server) *IgnoreClosedServer {
	return &IgnoreClosedServer{srv: srv}
}

// IgnoreClosedServer wraps a server and ignores [http.ErrServerClosed] returned by Start.
type IgnoreClosedServer struct {
	srv contract.Server
}

var _ contract.Server = (*IgnoreClosedServer)(nil)

// Start starts the wrapped server and returns nil for [http.ErrServerClosed].
func (s *IgnoreClosedServer) Start(ctx context.Context) error {
	err := s.srv.Start(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop stops the wrapped server.
func (s *IgnoreClosedServer) Stop(ctx context.Context) error {
	return s.srv.Stop(ctx)
}
