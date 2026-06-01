package otel

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

type options struct {
	provider log.LoggerProvider
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func newOptions(opts ...Option) *options {
	o := &options{
		provider: global.GetLoggerProvider(),
	}
	for _, opt := range opts {
		opt.apply(o)
	}
	return o
}

func WithLoggerProvider(provider log.LoggerProvider) Option {
	return optionFunc(func(o *options) {
		o.provider = provider
	})
}
