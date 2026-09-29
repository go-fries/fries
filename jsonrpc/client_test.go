package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type transportFunc func(context.Context, string, *Request) (*Response, error)

func (f transportFunc) Send(ctx context.Context, namespace string, request *Request) (*Response, error) {
	return f(ctx, namespace, request)
}

type codecStub struct {
	marshal   func(any) ([]byte, error)
	unmarshal func([]byte, any) error
}

func (c *codecStub) Marshal(value any) ([]byte, error) {
	return c.marshal(value)
}

func (c *codecStub) Unmarshal(data []byte, value any) error {
	return c.unmarshal(data, value)
}

type staticIDGenerator struct {
	id *ID
}

func (g staticIDGenerator) Generate() *ID {
	return g.id
}

func TestNewClientOptions(t *testing.T) {
	transport := transportFunc(func(context.Context, string, *Request) (*Response, error) {
		return &Response{}, nil
	})
	customCodec := &codecStub{
		marshal: func(any) ([]byte, error) { return nil, nil },
		unmarshal: func([]byte, any) error {
			return nil
		},
	}
	idGenerator := staticIDGenerator{id: NewID("fixed-id")}
	middleware := func(next Handler) Handler { return next }

	clientWithOptions := NewClient(
		transport,
		WithMiddlewares(middleware),
		WithIDGenerator(idGenerator),
		WithCodec(customCodec),
	).(*client)

	assert.Same(t, customCodec, clientWithOptions.codec)
	assert.Equal(t, idGenerator, clientWithOptions.idGenerator)
	assert.Len(t, clientWithOptions.middlewares, 1)

	clientWithOptions.Use(middleware)
	assert.Len(t, clientWithOptions.middlewares, 2)

	namespaced := clientWithOptions.Namespace("users").(*client)
	assert.Equal(t, "users", namespaced.namespace)
	assert.Empty(t, clientWithOptions.namespace)
	assert.NotNil(t, namespaced.transport)

	clientWithDefaults := NewClient(transport).(*client)
	assert.Equal(t, DefaultCodec, clientWithDefaults.codec)
	assert.Equal(t, DefaultIDGenerator, clientWithDefaults.idGenerator)
	assert.Empty(t, clientWithDefaults.middlewares)
}

func TestWithMiddlewaresCopiesInput(t *testing.T) {
	middlewares := []Middleware{func(next Handler) Handler { return next }}
	option := WithMiddlewares(middlewares...)
	middlewares[0] = nil

	client := NewClient(nil, option).(*client)
	require.Len(t, client.middlewares, 1)
	assert.NotNil(t, client.middlewares[0])
}

func TestClientNamespaceMiddlewaresAreIndependent(t *testing.T) {
	var calls []string
	transport := transportFunc(func(_ context.Context, namespace string, _ *Request) (*Response, error) {
		calls = append(calls, "transport:"+namespace)
		return &Response{Result: json.RawMessage(`null`)}, nil
	})
	parent := NewClient(transport).(*client)
	parent.middlewares = make([]Middleware, 0, 6)
	parent.Use(traceMiddleware("first", &calls), traceMiddleware("second", &calls), traceMiddleware("third", &calls))

	first := parent.Namespace("first").(*client)
	second := parent.Namespace("second").(*client)
	first.Use(traceMiddleware("first-only", &calls))
	second.Use(traceMiddleware("second-only", &calls))
	parent.Use(traceMiddleware("parent-only", &calls))

	for _, tt := range []struct {
		name   string
		client Client
		want   []string
	}{
		{"first", first, []string{"first", "second", "third", "first-only", "transport:first"}},
		{"second", second, []string{"first", "second", "third", "second-only", "transport:second"}},
		{"parent", parent, []string{"first", "second", "third", "parent-only", "transport:"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls = nil
			var result any
			_, err := tt.client.Invoke(t.Context(), &result, "method")
			require.NoError(t, err)
			assert.Equal(t, tt.want, calls)
		})
	}
}

func TestCombineMiddlewaresDoesNotModifyClientBacking(t *testing.T) {
	var calls []string
	original := make([]Middleware, 1, 2)
	original[0] = traceMiddleware("client", &calls)
	original[:2][1] = traceMiddleware("reserved", &calls)
	ctx := ContextWithMiddlewares(t.Context(), traceMiddleware("request", &calls))
	combined := combineMiddlewares(ctx, original)

	final := func(context.Context, string, *Request) (*Response, error) { return &Response{}, nil }
	_, err := chain(combined...)(final)(ctx, "", &Request{})
	require.NoError(t, err)
	assert.Equal(t, []string{"client", "request"}, calls)

	calls = nil
	_, err = chain(original[:2]...)(final)(ctx, "", &Request{})
	require.NoError(t, err)
	assert.Equal(t, []string{"client", "reserved"}, calls)
}

func traceMiddleware(name string, calls *[]string) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, namespace string, req *Request) (*Response, error) {
			*calls = append(*calls, name)
			return next(ctx, namespace, req)
		}
	}
}

func TestClientInvoke(t *testing.T) {
	id := NewID("fixed-id")
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
	transport := transportFunc(func(_ context.Context, namespace string, request *Request) (*Response, error) {
		calls = append(calls, "transport")
		assert.Equal(t, "users", namespace)
		assert.Equal(t, ProtocolVersion, request.JSONRPC)
		assert.Equal(t, "find", request.Method)
		assert.Same(t, id, request.ID)
		assert.JSONEq(t, `[123,"active"]`, string(request.Params))
		return &Response{
			JSONRPC: ProtocolVersion,
			Result:  json.RawMessage(`{"name":"Alice"}`),
			ID:      id,
		}, nil
	})
	client := NewClient(
		transport,
		WithIDGenerator(staticIDGenerator{id: id}),
		WithMiddlewares(middleware("client")),
	).Namespace("users")
	ctx := ContextWithMiddlewares(t.Context(), middleware("context"))
	var result struct {
		Name string `json:"name"`
	}

	resp, err := client.Invoke(ctx, &result, "find", 123, "active")

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Alice", result.Name)
	assert.Equal(t, []string{
		"client:before",
		"context:before",
		"transport",
		"context:after",
		"client:after",
	}, calls)
}

func TestClientInvokeConcurrentContextMiddlewares(t *testing.T) {
	const requests = 32
	type requestKey struct{}
	client := NewClient(transportFunc(func(context.Context, string, *Request) (*Response, error) {
		return &Response{Result: json.RawMessage(`null`)}, nil
	})).(*client)
	client.middlewares = make([]Middleware, 1, requests+1)
	client.middlewares[0] = func(next Handler) Handler { return next }

	start := make(chan struct{})
	errs := make(chan error, requests)
	for i := range requests {
		go func() {
			<-start
			ctx := context.WithValue(t.Context(), requestKey{}, i)
			ctx = ContextWithMiddlewares(ctx, func(next Handler) Handler {
				return func(ctx context.Context, namespace string, req *Request) (*Response, error) {
					if ctx.Value(requestKey{}) != i {
						return nil, fmt.Errorf("middleware %d ran for a different request", i)
					}
					return next(ctx, namespace, req)
				}
			})
			var result any
			_, err := client.Invoke(ctx, &result, "method")
			errs <- err
		}()
	}
	close(start)
	for range requests {
		assert.NoError(t, <-errs)
	}
}

func TestClientInvokeErrors(t *testing.T) {
	sentinel := errors.New("sentinel")

	t.Run("marshal", func(t *testing.T) {
		transportCalled := false
		client := NewClient(
			transportFunc(func(context.Context, string, *Request) (*Response, error) {
				transportCalled = true
				return nil, nil
			}),
			WithCodec(&codecStub{
				marshal: func(any) ([]byte, error) { return nil, sentinel },
				unmarshal: func([]byte, any) error {
					return nil
				},
			}),
		)

		resp, err := client.Invoke(t.Context(), nil, "method")

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, sentinel)
		assert.False(t, transportCalled)
	})

	t.Run("transport", func(t *testing.T) {
		wantResp := &Response{}
		client := NewClient(transportFunc(func(context.Context, string, *Request) (*Response, error) {
			return wantResp, sentinel
		}))

		resp, err := client.Invoke(t.Context(), nil, "method")

		assert.Same(t, wantResp, resp)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("rpc response", func(t *testing.T) {
		rpcErr := &Error{Code: -32601, Message: "method not found"}
		wantResp := &Response{Error: rpcErr}
		client := NewClient(transportFunc(func(context.Context, string, *Request) (*Response, error) {
			return wantResp, nil
		}))

		resp, err := client.Invoke(t.Context(), nil, "method")

		assert.Same(t, wantResp, resp)
		assert.Same(t, rpcErr, err)
	})

	t.Run("unmarshal", func(t *testing.T) {
		wantResp := &Response{Result: json.RawMessage(`"result"`)}
		client := NewClient(
			transportFunc(func(context.Context, string, *Request) (*Response, error) {
				return wantResp, nil
			}),
			WithCodec(&codecStub{
				marshal: func(any) ([]byte, error) { return []byte("[]"), nil },
				unmarshal: func([]byte, any) error {
					return sentinel
				},
			}),
		)

		resp, err := client.Invoke(t.Context(), nil, "method")

		assert.Same(t, wantResp, resp)
		assert.ErrorIs(t, err, sentinel)
	})
}
