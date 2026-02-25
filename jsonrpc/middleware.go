package jsonrpc //nolint:revive

import "context"

// Handler is the type for JSON-RPC request handlers.
type Handler func(ctx context.Context, namespace string, req *Request) (*Response, error)

// Middleware defines a function to process middleware.
type Middleware func(next Handler) Handler

// chain applies a list of middlewares to a final handler.
//
// Example: it produces a handler like
//
//	mw1(mw2(mw3(final)))
func chain(middlewares ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// middlewareContextKey is used as the key for storing middlewares in context.
type middlewareContextKey struct{}

// ContextWithMiddlewares returns a new context with the provided middlewares attached.
//
// These middlewares will be applied to any requests made with this context,
// in addition to any middlewares configured on the Client.
func ContextWithMiddlewares(ctx context.Context, mws ...Middleware) context.Context {
	return context.WithValue(ctx, middlewareContextKey{}, mws)
}

// middlewaresFromContext retrieves middlewares from the context.
func middlewaresFromContext(ctx context.Context) []Middleware {
	if v := ctx.Value(middlewareContextKey{}); v != nil {
		if mws, ok := v.([]Middleware); ok {
			return mws
		}
	}
	return nil
}
