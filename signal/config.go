package signal

import "log/slog"

type config struct {
	logger *slog.Logger
}

// Option configures a [Router].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) {
	f(cfg)
}

// WithLogger configures runtime, handler failure, and overflow logs. A nil
// logger leaves the default logger unchanged.
func WithLogger(logger *slog.Logger) Option {
	return optionFunc(func(cfg *config) {
		if logger != nil {
			cfg.logger = logger
		}
	})
}

func newConfig(opts ...Option) config {
	cfg := config{logger: slog.Default()}
	for _, opt := range opts {
		if opt != nil {
			opt.apply(&cfg)
		}
	}
	return cfg
}
