package server

import (
	"context"
	"net/http"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

type Server struct {
	name string
	*http.Server
}

type Option func(*Server)

func WithName(name string) Option {
	return func(s *Server) {
		if name != "" {
			s.name = name
		}
	}
}

var _ transport.Server = (*Server)(nil)

func New(srv *http.Server, opts ...Option) *Server {
	s := &Server{
		name:   "HTTP",
		Server: srv,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	log.Infof("[%s] server listening on: %s", s.name, s.Addr)
	return s.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Infof("[%s] server stopping", s.name)
	return s.Shutdown(ctx)
}
