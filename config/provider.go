package config

import "context"

// Provider is a generic structure that represents a provider with a specific configuration.
// It provides functionalities to bootstrap a context with the configuration and to terminate the context.
type Provider[T any] struct {
	config T
}

// NewProvider creates and returns a new instance of the Provider with the given configuration.
// It is a constructor function that initializes the Provider with a generic configuration.
func NewProvider[T any](config T) *Provider[T] {
	return &Provider[T]{config: config}
}

// Bootstrap initializes and returns a new context with the provider's configuration.
// It takes the current context as input and merges it with the provider's configuration to create a new context.
// This function is typically used at the beginning of an application's lifecycle to set up the necessary configurations.
func (p *Provider[T]) Bootstrap(ctx context.Context) (context.Context, error) {
	return NewContext(ctx, p.config), nil
}

// Terminate is a placeholder function that simulates the termination process of a context.
func (p *Provider[T]) Terminate(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
