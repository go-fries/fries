package cached

import "context"

type textContextKey struct{}

func contextWithText(ctx context.Context, text string) context.Context {
	return context.WithValue(ctx, textContextKey{}, text)
}

func TextFromContext(ctx context.Context) (string, bool) {
	text, ok := ctx.Value(textContextKey{}).(string)
	return text, ok
}
