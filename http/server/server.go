package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-fries/fries/capability/v4"
)

// Server wraps a standard library HTTP server with Start and Stop methods.
type Server struct {
	name   string
	server *http.Server
	logger *slog.Logger
}

var _ capability.Server = (*Server)(nil)

func New(srv *http.Server, opts ...Option) *Server {
	cfg := newConfig(opts...)
	if srv == nil {
		srv = &http.Server{}
	}
	if cfg.addr != "" {
		srv.Addr = cfg.addr
	}

	s := &Server{
		name:   cfg.name,
		server: srv,
		logger: cfg.logger,
	}
	return s
}

func NewWithHandler(handler http.Handler, opts ...Option) *Server {
	srv := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	return New(srv, opts...)
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "["+s.name+"] server listening on: "+s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "["+s.name+"] server stopping")
	return s.server.Shutdown(ctx)
}
