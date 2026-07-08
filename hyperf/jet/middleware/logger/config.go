package logger

import "log/slog"

type config struct {
	logger *slog.Logger
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		logger: slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(cfg)
		}
	}
	return cfg
}

// Option configures logger middleware.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithLogger configures the slog logger used by logger middleware.
//
// If logger is nil, the option leaves the current logger unchanged.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(cfg *config) {
		if logger != nil {
			cfg.logger = logger
		}
	})
}
