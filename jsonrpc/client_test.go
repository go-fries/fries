package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
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
