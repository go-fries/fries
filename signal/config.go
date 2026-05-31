package signal

import (
	"context"
	"os"
)

// Option configures a Server.
type Option interface {
	apply(*config)
}

type config struct {
	handlers []Handler
	recovery RecoveryHandler
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithHandlers registers handlers when constructing a Server.
func WithHandlers(handlers ...Handler) Option {
	return optionFunc(func(cfg *config) {
		cfg.handlers = append(cfg.handlers, handlers...)
	})
}

// WithRecovery configures a panic recovery hook for handler execution.
func WithRecovery(handler RecoveryHandler) Option {
	return optionFunc(func(cfg *config) {
		if handler != nil {
			cfg.recovery = handler
		}
	})
}

// RecoveryHandler handles panics raised by signal handlers.
type RecoveryHandler func(context.Context, any, os.Signal, Handler)
