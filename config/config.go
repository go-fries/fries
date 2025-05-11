package config

import "context"

// configKey is a type used to store configuration in the context.
// It uses generics (T) so that it can work with any type of configuration.
type configKey[T any] struct{}

// NewContext creates and returns a new context that carries configuration information.
// This function takes an existing context (ctx) and a configuration object (config),
// and stores the configuration in the new context using configKey as the key.
func NewContext[T any](ctx context.Context, config T) context.Context {
	return context.WithValue(ctx, configKey[T]{}, config)
}

// FromContext retrieves the configuration object from the given context.
// It uses configKey as the key to fetch the configuration.
func FromContext[T any](ctx context.Context) (T, bool) {
	config, ok := ctx.Value(configKey[T]{}).(T)
	return config, ok
}
