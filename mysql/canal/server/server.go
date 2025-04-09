package server

import (
	"context"

	"github.com/go-fries/fries/contract/v3"
	canal2 "github.com/go-fries/fries/mysql/canal/v3"
)

type Server struct {
	canal *canal2.Canal
}

var _ contract.Server = (*Server)(nil)

func NewCanalServer(canal *canal2.Canal) *Server {
	return &Server{
		canal: canal,
	}
}

func (s *Server) Start(ctx context.Context) error {
	return s.Start(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	return s.Stop(ctx)
}
