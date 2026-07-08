package server

import (
	"context"
	"log/slog"
	"net/http"
)

// Server represents a startable and stoppable HTTP server.
type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

var _ Server = (*HTTPServer)(nil)

type HTTPServer struct {
	name   string
	server *http.Server
	logger *slog.Logger
}

func New(srv *http.Server, opts ...Option) *HTTPServer {
	cfg := newConfig(opts...)
	if srv == nil {
		srv = &http.Server{}
	}
	if cfg.addr != "" {
		srv.Addr = cfg.addr
	}

	s := &HTTPServer{
		name:   cfg.name,
		server: srv,
		logger: cfg.logger,
	}
	return s
}

func NewWithHandler(handler http.Handler, opts ...Option) *HTTPServer {
	srv := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}

	return New(srv, opts...)
}

func (s *HTTPServer) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "["+s.name+"] server listening on: "+s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "["+s.name+"] server stopping")
	return s.server.Shutdown(ctx)
}
