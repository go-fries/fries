package crontab

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/go-kratos/kratos/v2/log"
)

// Server adapts a cron scheduler to the Kratos transport server lifecycle.
type Server struct {
	cron *cron.Cron
}

// NewServer creates a Kratos server for c.
func NewServer(c *cron.Cron) *Server {
	return &Server{
		cron: c,
	}
}

// Cron returns the underlying cron scheduler.
func (s *Server) Cron() *cron.Cron {
	return s.cron
}

// Start runs the cron scheduler until Stop is called.
func (s *Server) Start(ctx context.Context) error {
	log.Context(ctx).Info("[Crontab] server starting")
	s.cron.Run()
	return nil
}

// Stop stops the cron scheduler and waits for running jobs to finish.
func (s *Server) Stop(ctx context.Context) error {
	log.Context(ctx).Info("[Crontab] server stopping")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.cron.Stop().Done():
		return nil
	}
}
