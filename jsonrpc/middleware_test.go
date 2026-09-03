package jsonrpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChain(t *testing.T) {
	var calls []string
	middleware := func(name string) Middleware {
		return func(next Handler) Handler {
			return func(ctx context.Context, namespace string, req *Request) (*Response, error) {
				calls = append(calls, name+":before")
				resp, err := next(ctx, namespace, req)
				calls = append(calls, name+":after")
				return resp, err
			}
		}
	}
	final := func(context.Context, string, *Request) (*Response, error) {
		calls = append(calls, "handler")
		return &Response{}, nil
	}

	resp, err := chain(middleware("first"), middleware("second"))(final)(t.Context(), "", &Request{})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, []string{
		"first:before",
		"second:before",
		"handler",
		"second:after",
		"first:after",
	}, calls)
}

func TestMiddlewaresFromContext(t *testing.T) {
	middleware := func(next Handler) Handler { return next }

	ctx := ContextWithMiddlewares(t.Context(), middleware)
	middlewares := middlewaresFromContext(ctx)
	require.Len(t, middlewares, 1)
	assert.NotNil(t, middlewares[0])
	assert.Nil(t, middlewaresFromContext(t.Context()))

	invalidCtx := context.WithValue(t.Context(), middlewareContextKey{}, "not middleware")
	assert.Nil(t, middlewaresFromContext(invalidCtx))
}
