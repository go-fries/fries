package signal

import (
	"log/slog"
)

// Option configures a Server.
type Option interface {
	apply(*config)
}

type config struct {
	handlers []Handler
	logger   *slog.Logger
}

func newConfig(opts ...Option) config {
	cfg := config{
		handlers: make([]Handler, 0),
		logger:   slog.Default(),
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
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
		cfg.handlers = appendHandlers(cfg.handlers, handlers...)
	})
}

// WithLogger configures the slog logger used by server logs.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(cfg *config) {
		if logger != nil {
			cfg.logger = logger
		}
	})
}
