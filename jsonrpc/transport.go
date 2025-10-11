package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type Transport interface {
	Send(ctx context.Context, namespace string, request *Request) (*Response, error)
}

var _ Transport = (*HTTPTransport)(nil)

type HTTPTransport struct {
	addr       string
	userAgent  string
	httpClient *http.Client
}

type HTTPTransportOption interface {
	apply(*HTTPTransport)
}

type httpTransportOptionFunc func(*HTTPTransport)

func (f httpTransportOptionFunc) apply(h *HTTPTransport) {
	f(h)
}

func WithHTTPTransportUserAgent(userAgent string) HTTPTransportOption {
	return httpTransportOptionFunc(func(h *HTTPTransport) {
		h.userAgent = userAgent
	})
}

func WithHTTPTransportClient(client *http.Client) HTTPTransportOption {
	return httpTransportOptionFunc(func(h *HTTPTransport) {
		h.httpClient = client
	})
}

func NewHTTPTransport(addr string, opts ...HTTPTransportOption) *HTTPTransport {
	h := &HTTPTransport{
		addr:       addr,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt.apply(h)
	}
	return h
}

func (h *HTTPTransport) Send(ctx context.Context, namespace string, request *Request) (*Response, error) {
	addr := h.addr
	if namespace != "" {
		addr = addr + "/" + namespace
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
