package server

import (
	"context"

	"github.com/go-fries/fries/mysql/canal/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

type Server struct {
	canal *canal.Canal
}

var _ transport.Server = (*Server)(nil)

func New(canal *canal.Canal) *Server {
	return &Server{
		canal: canal,
	}
}

func (s *Server) Start(ctx context.Context) error {
	log.Infof("[Canal] server starting")
	return s.canal.Start(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	log.Infof("[Canal] server stopping")
	return s.canal.Stop(ctx)
}
