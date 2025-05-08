package config

import "context"

type configKey[T any] struct{}

func NewContext[T any](ctx context.Context, config T) context.Context {
	return context.WithValue(ctx, configKey[T]{}, config)
}

func FromContext[T any](ctx context.Context) (T, bool) {
	config, ok := ctx.Value(configKey[T]{}).(T)
	return config, ok
}
