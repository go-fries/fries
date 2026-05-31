package signal

import "os"

// Option configures a Server.
type Option interface {
	apply(*config)
}

type config struct {
	handlers []Handler
	recovery func(any, os.Signal, Handler)
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
func WithRecovery(handler func(any, os.Signal, Handler)) Option {
	return optionFunc(func(cfg *config) {
		if handler != nil {
			cfg.recovery = handler
		}
	})
}
