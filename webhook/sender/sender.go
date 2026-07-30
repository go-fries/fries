package sender

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-fries/fries/webhook/v4"
)

// Message contains one outbound webhook message.
type Message struct {
	// ID is the stable message ID used across delivery attempts.
	ID string
	// Payload contains the exact bytes to sign and send. The caller must not
	// modify Payload until Send returns.
	Payload []byte
	// Header contains additional request headers. Sender clones Header and
	// overwrites the Standard Webhooks signature headers. The caller must not
	// modify Header until Send returns.
	Header http.Header
}

// Response contains an HTTP response received from a webhook endpoint.
//
// Callers must close Body when Response is non-nil.
type Response struct {
	*http.Response

	// RetryAfter is the delay requested by a valid Retry-After header. It is
	// zero when the header is absent, invalid, or already elapsed.
	RetryAfter time.Duration
}

// Successful reports whether StatusCode is between 200 and 299.
func (r *Response) Successful() bool {
	return r != nil &&
		r.Response != nil &&
		r.StatusCode >= http.StatusOK &&
		r.StatusCode < http.StatusMultipleChoices
}

// Sender performs signed HTTP webhook deliveries to one endpoint.
//
// A Sender is safe for concurrent use. Endpoint validators must also be safe
// for concurrent use.
type Sender struct {
	endpoint   url.URL
	signer     *webhook.Signer
	client     *http.Client
	validators []EndpointValidator
	now        func() time.Time
}

// New creates a Sender bound to endpoint and signer.
//
// HTTPS is required unless [WithInsecureHTTP] is supplied. Redirects are never
// followed. A nil signer returns [ErrNilSigner]. Invalid and insecure
// endpoints return [ErrInvalidEndpoint] and [ErrInsecureEndpoint],
// respectively.
func New(
	endpoint string,
	signer *webhook.Signer,
	options ...Option,
) (*Sender, error) {
	if signer == nil {
		return nil, ErrNilSigner
	}

	c := newConfig(options...)
	parsed, err := parseEndpoint(endpoint, c.allowHTTP)
	if err != nil {
		return nil, err
	}

	return &Sender{
		endpoint:   *parsed,
		signer:     signer,
		client:     c.newHTTPClient(),
		validators: append([]EndpointValidator(nil), c.validators...),
		now:        time.Now,
	}, nil
}

// Send signs message and performs one HTTP POST request.
//
// Transport failures are returned as errors. HTTP responses, including 4xx
// and 5xx responses, are returned as Response values with a nil error. The
// caller must close Response.Body. Send does not retain or modify message. A
// nil ctx returns [ErrInvalidContext].
func (s *Sender) Send(
	ctx context.Context,
	message Message,
) (*Response, error) {
	if ctx == nil {
		return nil, ErrInvalidContext
	}

	for _, validator := range s.validators {
		endpoint := s.endpoint
		if err := validator(ctx, &endpoint); err != nil {
			return nil, fmt.Errorf(
				"%w: endpoint validator: %w",
				ErrInvalidEndpoint,
				err,
			)
		}
	}

	signatureHeaders, err := s.signer.Sign(message.ID, message.Payload)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.endpoint.String(),
		bytes.NewReader(message.Payload),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"webhook/sender: create request: %w",
			err,
		)
	}
	// Prevent net/http from replaying a POST after a possible delivery. In
	// particular, Transport treats requests with an idempotency header and a
	// GetBody function as replayable.
	request.GetBody = nil

	request.Header = cloneHeader(message.Header)
	if request.Header.Get("content-type") == "" {
		request.Header.Set("content-type", "application/json")
	}
	for name, values := range signatureHeaders {
		request.Header.Del(name)
		request.Header[name] = append([]string(nil), values...)
	}

	//nolint:bodyclose // Response ownership is transferred to the caller.
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"webhook/sender: send request: %w",
			err,
		)
	}

	return &Response{
		Response:   response,
		RetryAfter: parseRetryAfter(response.Header.Get("retry-after"), s.now()),
	}, nil
}

func parseEndpoint(value string, allowHTTP bool) (*url.URL, error) {
	endpoint, err := url.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEndpoint, err)
	}
	if !endpoint.IsAbs() ||
		endpoint.Hostname() == "" ||
		endpoint.User != nil ||
		endpoint.Fragment != "" ||
		endpoint.Opaque != "" {
		return nil, ErrInvalidEndpoint
	}
	if port := endpoint.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return nil, ErrInvalidEndpoint
		}
	}

	scheme := strings.ToLower(endpoint.Scheme)
	endpoint.Scheme = scheme
	switch scheme {
	case "https":
		return endpoint, nil
	case "http":
		if !allowHTTP {
			return nil, ErrInsecureEndpoint
		}
		return endpoint, nil
	default:
		return nil, ErrInvalidEndpoint
	}
}

func cloneHeader(source http.Header) http.Header {
	target := make(http.Header, len(source))
	for name, values := range source {
		canonicalName := http.CanonicalHeaderKey(name)
		target[canonicalName] = append(target[canonicalName], values...)
	}
	return target
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	seconds, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if seconds <= 0 || seconds > math.MaxInt64/int64(time.Second) {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	date, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := date.Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
}
