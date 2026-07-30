package sender

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 30 * time.Second

type config struct {
	client     *http.Client
	timeout    time.Duration
	allowHTTP  bool
	validators []EndpointValidator
}

// Option configures a [Sender].
type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

// WithHTTPClient sets the HTTP client used for delivery. The Sender copies the
// client and disables redirects without modifying the supplied value.
//
// A nil client leaves the current client unchanged.
func WithHTTPClient(client *http.Client) Option {
	return optionFunc(func(c *config) {
		if client != nil {
			c.client = client
		}
	})
}

// WithTimeout sets the total HTTP request timeout. Non-positive values are
// ignored.
func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *config) {
		if timeout > 0 {
			c.timeout = timeout
		}
	})
}

// WithInsecureHTTP allows an endpoint to use HTTP instead of HTTPS.
//
// Use this only for trusted development or private-network endpoints whose
// transport security is provided elsewhere.
func WithInsecureHTTP() Option {
	return optionFunc(func(c *config) {
		c.allowHTTP = true
	})
}

// EndpointValidator validates an endpoint immediately before each delivery.
//
// A validator may enforce an application-specific hostname allowlist or other
// outbound policy. It must be safe for concurrent use.
type EndpointValidator func(context.Context, *url.URL) error

// WithEndpointValidator adds an endpoint validator. Nil validators are
// ignored.
func WithEndpointValidator(validator EndpointValidator) Option {
	return optionFunc(func(c *config) {
		if validator != nil {
			c.validators = append(c.validators, validator)
		}
	})
}

func newConfig(options ...Option) config {
	c := config{
		client: &http.Client{},
	}
	for _, option := range options {
		if option != nil {
			option.apply(&c)
		}
	}
	return c
}

func (c config) newHTTPClient() *http.Client {
	client := *c.client
	switch {
	case c.timeout > 0:
		client.Timeout = c.timeout
	case client.Timeout <= 0:
		client.Timeout = defaultTimeout
	}
	client.CheckRedirect = func(
		_ *http.Request,
		_ []*http.Request,
	) error {
		return http.ErrUseLastResponse
	}
	return &client
}
