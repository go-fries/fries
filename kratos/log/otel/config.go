package otel

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type config struct {
	provider   log.LoggerProvider
	version    string
	schemaURL  string
	attributes []attribute.KeyValue
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(c *config) {
		c.provider = provider
	})
}

func WithVersion(version string) Option {
	return optionFunc(func(c *config) {
		c.version = version
	})
}

func WithSchemaURL(schemaURL string) Option {
	return optionFunc(func(c *config) {
		c.schemaURL = schemaURL
	})
}

func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(c *config) {
		c.attributes = append(c.attributes, attributes...)
	})
}

func newConfig(opts ...Option) *config {
	cfg := &config{
		provider: global.GetLoggerProvider(),
		version:  Version(),
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	return cfg
}

func (c *config) newLogger(name string) log.Logger {
	opts := []log.LoggerOption{
		log.WithInstrumentationVersion(c.version),
	}
	if c.schemaURL != "" {
		opts = append(opts, log.WithSchemaURL(c.schemaURL))
	}
	if len(c.attributes) > 0 {
		opts = append(opts, log.WithInstrumentationAttributes(c.attributes...))
	}

	return c.provider.Logger(
		name,
		opts...,
	)
}
