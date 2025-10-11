package jsonrpc

import (
	"context"

	"github.com/go-fries/fries/codec/json/v3"
	"github.com/go-fries/fries/codec/v3"
)

type Client interface {
	Namespace(name string) Client
	Call(ctx context.Context, result any, method string, args ...any) (*Response, error)
}

type client struct {
	namespace string

	transport   Transport
	idGenerator IDGenerator
	codec       codec.Codec
}

func NewClient(transport Transport) Client {
	return &client{
		transport:   transport,
		idGenerator: NewIDGenerator(),
		codec:       json.Codec,
	}
}

func (c *client) Namespace(name string) Client {
	nc := new(client)
	*nc = *c
	nc.namespace = name
	return nc
}

func (c *client) Call(ctx context.Context, result any, method string, args ...any) (*Response, error) {
	bytes, err := c.codec.Marshal(args)
	if err != nil {
		return nil, err
	}

	req := &Request{
		JSONRPC: JSONRPCVersion,
		Method:  method,
		Params:  bytes,
		ID:      c.idGenerator.Generate(),
	}
	resp, err := c.transport.Send(ctx, c.namespace, req)
	if err != nil {
		return resp, err
	}
	if resp.Error != nil {
		return resp, resp.Error
	}

	return resp, c.codec.Unmarshal(resp.Result, result)
}
