package server

import (
	"context"
	"sync"

	"github.com/go-fries/fries/queue/v3"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// Server adapts a queue worker to the Kratos transport server lifecycle.
type Server struct {
	worker *queue.Worker

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
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
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	s.mu.Lock()
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	log.Infof("[Queue] server starting")
	err := s.worker.Run(runCtx)

	s.mu.Lock()
	if s.done == done {
		s.cancel = nil
	}
	close(done)
	s.mu.Unlock()

	return err
}

// Stop stops the queue worker and waits until it exits or ctx is canceled.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	if cancel == nil || done == nil {
		return nil
	}

	log.Infof("[Queue] server stopping")
	cancel()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
