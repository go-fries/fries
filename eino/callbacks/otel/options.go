package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type options struct {
	tp trace.TracerProvider
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithTracerProvider(tp trace.TracerProvider) Option {
	return optionFunc(func(h *options) {
		h.tp = tp
	})
}

func newOptions(opts ...Option) *options {
	o := &options{
		tp: otel.GetTracerProvider(),
	}
	for _, opt := range opts {
		opt.apply(o)
	}
	return o
}
