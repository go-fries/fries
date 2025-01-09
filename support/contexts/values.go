package contexts

import "context"

func Value[T any](ctx context.Context, key any) (T, bool) {
	v, ok := ctx.Value(key).(T)
	return v, ok
}

func MustValue[T any](ctx context.Context, key any) T {
	v, ok := Value[T](ctx, key)
	if !ok {
		panic("contexts: key not exists")
	}
	return v
}
