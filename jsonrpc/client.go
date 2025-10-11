package jsonrpc

import (
	"context"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
)

var DefaultCodec = json.Codec

// Client represents a JSON-RPC client that can invoke remote methods.
// It supports namespacing to organize method calls and uses a Transport
// to send requests and receive responses.
type Client interface {
	// Use adds one or more middlewares to the client. These middlewares will be applied
	// to all requests made by the client.
	//
	// Example:
	//	client := NewClient(transport)
	//	client.Use(loggingMiddleware, authMiddleware)
	Use(middlewares ...Middleware)

	// Namespace returns a new Client instance scoped to the specified namespace.
	// This allows organizing and isolating method calls under different namespaces.
	//
	// Example:
	//	client := NewClient(transport)
	//	nsClient := client.Namespace("myNamespace")
	//	resp, err := nsClient.Invoke(ctx, &result, "methodName", arg1, arg2)
	//
	// The original client remains unchanged and can be used to create other namespace-scoped clients.
	//
	// Note: The namespace is typically a string that groups related methods together,
	// such as a service name or module identifier.
	//
	// Concurrency Note:
	// Returns a new Client instance with the specified namespace. The method does not support concurrency
	// safety; ensure that the returned Client is used in a single-threaded context or manage synchronization externally.
	Namespace(name string) Client

	// Invoke invokes a remote method with the given arguments and populates the result.
	// The result parameter should be a pointer to the expected result type.
	// It returns the full Response and an error if the invocation fails.
	//
	// Example:
	//	var result MyResultType
	//	resp, err := client.Invoke(ctx, &result, "methodName", arg1, arg2)
	//
	// If the remote method returns an error, it will be contained in resp.Error.
	// If the invocation itself fails (e.g., network error), err will be non-nil.
	//
	// Note: The result parameter must be a pointer type to allow unmarshalling of the response.
	Invoke(ctx context.Context, result any, method string, args ...any) (*Response, error)
}

// client is the concrete implementation of the Client interface.
type client struct {
	// namespace is an optional string that scopes the client's method calls.
	namespace string

	// transport is the Transport used to send requests and receive responses.
	transport Transport

	// middlewares is a slice of Middleware functions that will be applied to each request.
	middlewares []Middleware

	// idGenerator generates unique IDs for each request.
	idGenerator IDGenerator

	// codec is used to marshal and unmarshal request and response payloads.
	codec codec.Codec
}

// Option defines a configuration option for the Client.
type Option interface {
	apply(*client)
}

// optionFunc is a helper type to implement Option using functions.
type optionFunc func(*client)

func (f optionFunc) apply(c *client) {
	f(c)
}

// WithMiddlewares adds one or more middlewares to the client.
func WithMiddlewares(middlewares ...Middleware) Option {
	return optionFunc(func(cl *client) {
		cl.middlewares = append(cl.middlewares, middlewares...)
	})
}

// WithIDGenerator sets a custom IDGenerator for the client.
func WithIDGenerator(g IDGenerator) Option {
	return optionFunc(func(cl *client) {
		cl.idGenerator = g
	})
}

// WithCodec sets a custom codec for the client.
func WithCodec(c codec.Codec) Option {
	return optionFunc(func(cl *client) {
		cl.codec = c
	})
}

// NewClient creates a new Client with the given Transport.
func NewClient(transport Transport, opts ...Option) Client {
	c := &client{
		transport:   transport,
		middlewares: make([]Middleware, 0),
		idGenerator: DefaultIDGenerator,
		codec:       DefaultCodec,
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Use adds one or more middlewares to the client. These middlewares will be applied
// to all requests made by the client.
func (c *client) Use(middlewares ...Middleware) {
	c.middlewares = append(c.middlewares, middlewares...)
}

// Namespace returns a new Client instance scoped to the specified namespace.
func (c *client) Namespace(name string) Client {
	nc := new(client)
	*nc = *c
	nc.namespace = name
	return nc
}

// Invoke invokes a remote method with the given arguments and populates the result.
func (c *client) Invoke(ctx context.Context, result any, method string, args ...any) (*Response, error) {
	bytes, err := c.codec.Marshal(args)
	if err != nil {
		return nil, err
	}

	req := &Request{
		JSONRPC: Version,
		Method:  method,
		Params:  bytes,
		ID:      c.idGenerator.Generate(),
	}

	resp, err := chain(c.middlewares...)(c.transport.Send)(ctx, c.namespace, req)
	if err != nil {
		return resp, err
	}
	if resp.Error != nil {
		return resp, resp.Error
	}

	return resp, c.codec.Unmarshal(resp.Result, result)
}
