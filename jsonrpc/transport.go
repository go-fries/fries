package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Transport defines the interface for sending JSON-RPC requests and receiving responses.
type Transport interface {
	// Send sends a JSON-RPC request to the specified namespace and returns the response.
	// The namespace parameter allows scoping the request to a specific context or service.
	//
	// Parameters:
	//   - ctx: The context for managing request lifetime and cancellation.
	//   - namespace: A string representing the namespace to which the request is sent.
	//   - request: A pointer to the Request struct containing the JSON-RPC request details.
	//
	// Returns:
	//   - A pointer to the Response struct containing the JSON-RPC response.
	//   - An error if the request fails or if there is an issue with sending or receiving the response.
	//
	// Note: Implementations of this method should handle serialization of the request,
	// network communication, and deserialization of the response.
	Send(ctx context.Context, namespace string, request *Request) (*Response, error)
}

var _ Transport = (*HTTPTransport)(nil)

// HTTPTransport is an implementation of the Transport interface using HTTP as the transport protocol.
type HTTPTransport struct {
	addr       string
	userAgent  string
	httpClient *http.Client
}

// HTTPTransportOption defines a function type for configuring HTTPTransport options.
type HTTPTransportOption interface {
	// apply applies the option to the given HTTPTransport instance.
	apply(*HTTPTransport)
}

// httpTransportOptionFunc is a helper type to implement HTTPTransportOption using functions.
type httpTransportOptionFunc func(*HTTPTransport)

func (f httpTransportOptionFunc) apply(h *HTTPTransport) {
	f(h)
}

// WithHTTPTransportUserAgent sets the User-Agent header for the HTTPTransport.
func WithHTTPTransportUserAgent(userAgent string) HTTPTransportOption {
	return httpTransportOptionFunc(func(h *HTTPTransport) {
		h.userAgent = userAgent
	})
}

// WithHTTPTransportClient sets a custom http.Client for the HTTPTransport.
func WithHTTPTransportClient(client *http.Client) HTTPTransportOption {
	return httpTransportOptionFunc(func(h *HTTPTransport) {
		h.httpClient = client
	})
}

// NewHTTPTransport creates a new HTTPTransport with the specified address and options.
// The addr parameter specifies the base URL for the JSON-RPC server.
// Additional options can be provided to customize the transport behavior.
func NewHTTPTransport(addr string, opts ...HTTPTransportOption) *HTTPTransport {
	h := &HTTPTransport{
		addr:       strings.TrimSuffix(addr, "/"),
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt.apply(h)
	}
	return h
}

// Send sends a JSON-RPC request to the specified namespace and returns the response.
// It constructs an HTTP POST request with the JSON-encoded request body and processes the response.
func (h *HTTPTransport) Send(ctx context.Context, namespace string, request *Request) (*Response, error) {
	addr := h.addr
	if namespace != "" {
		addr = addr + "/" + strings.TrimPrefix(namespace, "/")
	}

	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(request); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, io.NopCloser(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.userAgent != "" {
		httpReq.Header.Set("User-Agent", h.userAgent)
	}

	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close() //nolint:errcheck

	var resp Response
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
