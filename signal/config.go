package signal

import (
	"context"
	"os"

	"github.com/go-kratos/kratos/v2/log"
)

// Option configures a Server.
type Option interface {
	apply(*config)
}

type config struct {
	handlers []Handler
	recovery RecoveryHandler
}

func newConfig(opts ...Option) config {
	cfg := config{
		handlers: make([]Handler, 0),
		recovery: defaultRecovery,
	}

	for _, opt := range opts {
		opt.apply(&cfg)
	}

	return cfg
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
type RecoveryHandler func(context.Context, os.Signal, Handler, any)

// DefaultRecovery logs panics raised by signal handlers.
func DefaultRecovery(ctx context.Context, sig os.Signal, handler Handler, recovered any) {
	defaultRecovery(ctx, sig, handler, recovered)
}

var defaultRecovery RecoveryHandler = func(ctx context.Context, sig os.Signal, _ Handler, recovered any) {
	log.Context(ctx).Errorf("[Signal] handler panic (%s): %v", sig, recovered)
}
