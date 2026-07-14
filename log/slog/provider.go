// Package slog provides lifecycle integration for the standard library's log/slog package.
package slog

import (
	"context"
	"log/slog"
)

// Provider configures the process-wide default slog logger during bootstrap.
type Provider struct {
	logger *slog.Logger
}

// NewProvider creates a Provider backed by logger.
func NewProvider(logger *slog.Logger) *Provider {
	return &Provider{logger: logger}
}

// Bootstrap sets the process-wide default slog logger.
func (p *Provider) Bootstrap(ctx context.Context) (context.Context, error) {
	slog.SetDefault(p.logger)
	return ctx, nil
}

// Shutdown leaves the process-wide default slog logger unchanged.
func (p *Provider) Shutdown(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
