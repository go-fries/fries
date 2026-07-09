package server

import "log/slog"

type config struct {
	addr   string
	name   string
	logger *slog.Logger
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		name:   "HTTP",
		logger: slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(cfg)
		}
	}
	return cfg
}

// Option configures a Server.
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithAddr configures the listen address.
func WithAddr(addr string) Option {
	return optionFunc(func(cfg *config) {
		cfg.addr = addr
	})
}

// WithName configures the server name used by lifecycle logs.
func WithName(name string) Option {
	return optionFunc(func(cfg *config) {
		if name != "" {
			cfg.name = name
		}
	})
}

// WithLogger configures the slog logger used by server lifecycle logs.
//
// If logger is nil, the option leaves the current logger unchanged.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(cfg *config) {
		if logger != nil {
			cfg.logger = logger
		}
	})
}
