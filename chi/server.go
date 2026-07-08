package chi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	server *http.Server
	logger *slog.Logger
}

type server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

var _ server = (*Server)(nil)

func NewServer(c *chi.Mux, opts ...Option) *Server {
	cfg := newConfig(opts...)
	if c == nil {
		c = chi.NewRouter()
	}

	srv := &Server{
		logger: cfg.logger,
	}

	srv.server = &http.Server{
		Addr:    cfg.addr,
		Handler: c,
	}

	return srv
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "[go-chi] server listening on: "+s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "[go-chi] server stopping")
	return s.server.Shutdown(ctx)
}
