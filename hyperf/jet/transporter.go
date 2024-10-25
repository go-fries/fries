package jet

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var (
	ErrHTTPTransporterAddrIsRequired     = errors.New("jet/transporter: Addr is required")
	ErrorHTTPTransporterClientIsRequired = errors.New("jet/transporter: client is required")
)

type Transporter interface {
	Send(ctx context.Context, data []byte) ([]byte, error)
}

// --------------------------------------------------------
// HTTPTransporter implementation
// --------------------------------------------------------

// HTTPTransporter is a http transporter
type HTTPTransporter struct {
	Addr string
	*http.Client
}

type HTTPTransporterOption func(*HTTPTransporter)

func WithHTTPTransporterAddr(addr string) HTTPTransporterOption {
	return func(t *HTTPTransporter) {
		t.Addr = addr
	}
}

func WithHTTPTransporterClient(client *http.Client) HTTPTransporterOption {
	return func(t *HTTPTransporter) {
		t.Client = client
	}
}

func NewHTTPTransporter(opts ...HTTPTransporterOption) (*HTTPTransporter, error) {
	transport := &HTTPTransporter{
		Client: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(transport)
	}

	// validate
	if transport.Addr == "" {
		return nil, ErrHTTPTransporterAddrIsRequired
	}
	if transport.Client == nil {
		return nil, ErrorHTTPTransporterClientIsRequired
	}

	return transport, nil
}

func (t *HTTPTransporter) Send(ctx context.Context, data []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Addr, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	response, err := t.Client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close() // nolint:errcheck

	// check response status code
	if isHTTPTransporterServerFailed(response) {
		return nil, &HTTPTransporterServerError{
			StatusCode: response.StatusCode,
			Message:    response.Status,
			Err:        fmt.Errorf("failed to send request"),
		}
	}

	return io.ReadAll(response.Body)
}

func isHTTPTransporterServerFailed(resp *http.Response) bool {
	return resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest
}

// IsHTTPTransporterServerError reports whether err was created by HTTPTransporterServerError.
func IsHTTPTransporterServerError(err error) bool {
	var target *HTTPTransporterServerError
	return errors.As(err, &target)
}

type HTTPTransporterServerError struct {
	StatusCode int
	Message    string
	Err        error
}

func (e *HTTPTransporterServerError) Error() string {
	return fmt.Sprintf("jet/transporter: server error, status code: %d, message: %s, error: %v", e.StatusCode, e.Message, e.Err) //nolint:lll
}

func (e *HTTPTransporterServerError) Unwrap() error {
	return e.Err
}
