package server

import (
	"context"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

type Server struct {
	*http.Server
}

var _ transport.Server = (*Server)(nil)

func New(srv *http.Server) *Server {
	return &Server{
		Server: srv,
	}
}

type HTTPServerOption func(*http.Server)

func WithHTTPServerAddr(addr string) HTTPServerOption {
	return func(s *http.Server) {
		s.Addr = addr
	}
}

func NewWithHandler(handler http.Handler, opts ...HTTPServerOption) *Server {
	srv := &http.Server{
		Handler: handler,
		Addr:    ":8080",
	}
	for _, opt := range opts {
		opt(srv)
	}

	return New(srv)
}

func (s *Server) Start(_ context.Context) error {
	log.Infof("[HTTP] server listening on: %s", s.Addr)
	return s.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[HTTP] server stopping")
	return s.Shutdown(ctx)
}
