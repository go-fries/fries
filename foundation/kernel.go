package foundation

import (
	"context"
	"slices"
	"time"
)

const defaultTerminateTimeout = 5 * time.Second

type Handler interface {
	Handle(ctx context.Context) error
}

type HandlerFunc func(ctx context.Context) error

func (f HandlerFunc) Handle(ctx context.Context) error {
	return f(ctx)
}

type Kernel struct {
	handler          Handler
	providers        []Provider
	terminateTimeout time.Duration
}

type Option func(*Kernel)

func WithHandler(handler Handler) Option {
	return func(k *Kernel) {
		k.handler = handler
	}
}

func WithProviders(providers ...Provider) Option {
	return func(k *Kernel) {
		k.providers = append(k.providers, providers...)
	}
}

func WithTerminateTimeout(timeout time.Duration) Option {
	return func(k *Kernel) {
		k.terminateTimeout = timeout
	}
}

func NewKernel(opts ...Option) *Kernel {
	kernel := &Kernel{}
	for _, opt := range opts {
		opt(kernel)
	}
	kernel.init()
	return kernel
}

func (k *Kernel) init() {
	if k.terminateTimeout <= 0 {
		k.terminateTimeout = defaultTerminateTimeout
	}
}

func (k *Kernel) Register(providers ...Provider) {
	k.providers = append(k.providers, providers...)
}

func (k *Kernel) bootstrap(ctx context.Context) (context.Context, error) {
	var err error
	for _, provider := range k.providers {
		if ctx, err = provider.Bootstrap(ctx); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (k *Kernel) Run(ctx context.Context) (err error) {
	if ctx, err = k.bootstrap(ctx); err != nil {
		return err
	}

	defer func() {
		ctx, cancel := context.WithTimeout(ctx, k.terminateTimeout)
		defer cancel()

		if _, err = k.terminate(ctx); err != nil {
			return
		}
	}()

	if k.handler != nil {
		return k.handler.Handle(ctx)
	}

	return nil
}

func (k *Kernel) terminate(ctx context.Context) (context.Context, error) {
	var err error
	for _, provider := range slices.Backward(k.providers) {
		if ctx, err = provider.Terminate(ctx); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
