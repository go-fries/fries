package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Transport interface {
	Send(ctx context.Context, namespace string, request *Request) (*Response, error)
}

var _ Transport = (*HTTPTransport)(nil)

type HTTPTransport struct {
	addr       string
	httpClient *http.Client
}

func NewHTTPTransport(addr string) *HTTPTransport {
	return &HTTPTransport{
		addr:       addr,
		httpClient: http.DefaultClient,
	}
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

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
