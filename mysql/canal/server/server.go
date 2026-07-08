package server

import (
	"context"
	"log/slog"

	"github.com/go-fries/fries/mysql/canal/v4"
)

type Server struct {
	canal  *canal.Canal
	logger *slog.Logger
}

type server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

var _ server = (*Server)(nil)

func New(canal *canal.Canal, opts ...Option) *Server {
	cfg := newConfig(opts...)
	return &Server{
		canal:  canal,
		logger: cfg.logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.InfoContext(ctx, "[Canal] server starting")
	return s.canal.Start(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.InfoContext(ctx, "[Canal] server stopping")
	return s.canal.Stop(ctx)
}
