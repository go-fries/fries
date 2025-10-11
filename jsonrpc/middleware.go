package jsonrpc

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
