package otel

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider log.LoggerProvider
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		provider: global.GetLoggerProvider(),
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

func (c *config) newLogger(name string) log.Logger {
	return c.provider.Logger(
		name,
		log.WithInstrumentationVersion(Version()),
	)
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(c *config) {
		c.provider = provider
	})
}
