package jet

import "context"

type contextClientKey struct{}

// ContextWithClient returns a new Context that carries value.
func ContextWithClient(ctx context.Context, client *Client) context.Context {
	return context.WithValue(ctx, contextClientKey{}, client)
}

// ClientFromContext returns the Client value stored in ctx, if any.
func ClientFromContext(ctx context.Context) (*Client, bool) {
	client, ok := ctx.Value(contextClientKey{}).(*Client)
	return client, ok
}
