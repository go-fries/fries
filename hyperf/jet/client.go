package jet

import (
	"context"
	"errors"
)

var (
	ErrClientServiceIsRequired     = errors.New("jet/client: service is required")
	ErrClientTransporterIsRequired = errors.New("jet/client: transporter is required")
)

type Client struct {
	// Service name
	service string

	// Middlewares
	middlewares []Middleware

	// Transporter
	transporter Transporter

	// IDGenerator
	idGenerator IDGenerator

	// PathGenerator
	pathGenerator PathGenerator

	// Formatter
	formatter Formatter

	// Packer
	packer Packer
}

type Option func(*Client)

func WithMiddleware(m ...Middleware) Option {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, m...)
	}
}

func WithService(service string) Option {
	return func(c *Client) {
		c.service = service
	}
}

func WithTransporter(transporter Transporter) Option {
	return func(c *Client) {
		c.transporter = transporter
	}
}

func WithIDGenerator(generator IDGenerator) Option {
	return func(c *Client) {
		c.idGenerator = generator
	}
}

func WithPathGenerator(generator PathGenerator) Option {
	return func(c *Client) {
		c.pathGenerator = generator
	}
}

func WithFormatter(formatter Formatter) Option {
	return func(c *Client) {
		c.formatter = formatter
	}
}

func WithPacker(packer Packer) Option {
	return func(c *Client) {
		c.packer = packer
	}
}

func NewClient(opts ...Option) (*Client, error) {
	client := &Client{
		idGenerator:   DefaultIDGenerator,
		pathGenerator: DefaultPathGenerator,
		formatter:     DefaultFormatter,
		packer:        DefaultPacker,
	}
	for _, opt := range opts {
		opt(client)
	}

	// validate
	if client.service == "" {
		return nil, ErrClientServiceIsRequired
	}
	if client.transporter == nil {
		return nil, ErrClientTransporterIsRequired
	}

	return client, nil
}

func (c *Client) Invoke(ctx context.Context, method string, request any, response any, middlewares ...Middleware) (err error) { // nolint:lll
	handler := func(ctx context.Context, service string, method string, request any) (any, error) {
		err = c.invoke(ctx, service, method, request, response)
		return response, err
	}

	handler = Chain(append(c.middlewares, middlewares...)...)(handler)

	response, err = handler(ContextWithClient(ctx, c), c.service, method, request)
	return
}

func (c *Client) invoke(ctx context.Context, service, method string, request any, response any) error {
	params, err := c.packer.Pack(request)
	if err != nil {
		return err
	}

	req, err := c.formatter.FormatRequest(&RPCRequest{
		ID:     c.idGenerator.Generate(),
		Path:   c.pathGenerator.Generate(service, method),
		Params: params,
	})
	if err != nil {
		return err
	}

	resp, err := c.transporter.Send(ctx, req)
	if err != nil {
		return err
	}

	rpcResp, err := c.formatter.ParseResponse(resp)
	if err != nil {
		return err
	}

	return c.packer.Unpack(rpcResp.Result, response)
}

func (c *Client) Use(m ...Middleware) {
	c.middlewares = append(c.middlewares, m...)
}

func (c *Client) GetService() string {
	return c.service
}

func (c *Client) GetTransporter() Transporter {
	return c.transporter
}

func (c *Client) GetIDGenerator() IDGenerator {
	return c.idGenerator
}

func (c *Client) GetPathGenerator() PathGenerator {
	return c.pathGenerator
}

func (c *Client) GetFormatter() Formatter {
	return c.formatter
}

func (c *Client) GetPacker() Packer {
	return c.packer
}
