package gin

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
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

func NewServer(e *gin.Engine, opts ...Option) *Server {
	cfg := newConfig(opts...)
	if e == nil {
		e = gin.New()
	}

	srv := &Server{
		logger: cfg.logger,
	}
	srv.server = &http.Server{
		Addr:    cfg.addr,
		Handler: e,
	}

	return srv
}

func (s *Server) Start(_ context.Context) error {
	s.logger.Info("[GIN] server listening on: " + s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("[GIN] server stopping")
	return s.server.Shutdown(ctx)
}
