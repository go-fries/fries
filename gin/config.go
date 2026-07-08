package gin

import "log/slog"

type config struct {
	addr   string
	logger *slog.Logger
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		addr:   ":8080",
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
