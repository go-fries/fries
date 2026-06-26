package server

import (
	"context"

	"github.com/go-fries/fries/queue/v4"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// Server adapts a queue worker to the Kratos transport server lifecycle.
type Server struct {
	worker *queue.Worker
}

var _ transport.Server = (*Server)(nil)

// New creates a Kratos server for worker.
func New(worker *queue.Worker) *Server {
	return &Server{
		worker: worker,
	}
}

// Start runs the queue worker until Stop is called or the worker returns an error.
func (s *Server) Start(ctx context.Context) error {
	log.Infof("[Queue] server starting")
	return s.worker.Run(ctx)
}

// Stop stops polling for new tasks and waits for in-flight tasks to finish.
func (s *Server) Stop(ctx context.Context) error {
	log.Infof("[Queue] server stopping")
	return s.worker.Stop(ctx)
}
