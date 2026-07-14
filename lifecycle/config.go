package lifecycle

import "time"

const defaultShutdownTimeout = 5 * time.Second

type config struct {
	providers       []Provider
	shutdownTimeout time.Duration
}

// Option configures a [Runner].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithProviders appends providers in bootstrap order. Providers shut down in
// the reverse order.
//
// Nil providers are ignored.
func WithProviders(providers ...Provider) Option {
	return optionFunc(func(c *config) {
		for _, provider := range providers {
			if provider != nil {
				c.providers = append(c.providers, provider)
			}
		}
	})
}

// WithShutdownTimeout sets the total time available to shut down all providers.
// The timeout applies to normal shutdown and startup rollback.
//
// Values that are not positive leave the current timeout unchanged.
func WithShutdownTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.shutdownTimeout = timeout
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		shutdownTimeout: defaultShutdownTimeout,
	}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}
