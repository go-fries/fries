package otel

import "go.opentelemetry.io/otel/trace"

type Option interface {
	apply(*Handler)
}

type optionFunc func(*Handler)

func (f optionFunc) apply(h *Handler) {
	f(h)
}

func WithTracerProvider(tp trace.TracerProvider) Option {
	return optionFunc(func(h *Handler) {
		h.tp = tp
	})
}
