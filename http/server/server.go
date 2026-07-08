package server

import (
	"context"
	"log/slog"
	"net/http"
)

type Server struct {
	name   string
	server *http.Server
	logger *slog.Logger
}

type server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

var _ server = (*Server)(nil)

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

func (s *Server) Start(_ context.Context) error {
	s.logger.Info("[" + s.name + "] server listening on: " + s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("[" + s.name + "] server stopping")
	return s.server.Shutdown(ctx)
}
