package sender

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-fries/fries/webhook/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSenderSend(t *testing.T) {
	t.Parallel()

	secret := mustSecret(t)
	verifier, err := webhook.NewVerifier(secret)
	require.NoError(t, err)

	client := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			body, readErr := io.ReadAll(request.Body)
			require.NoError(t, readErr)

			metadata, verifyErr := verifier.Verify(request.Header, body)
			require.NoError(t, verifyErr)
			assert.Equal(t, "msg_123", metadata.ID)
			assert.Equal(t, `{"type":"order.created"}`, string(body))
			assert.Equal(
				t,
				"application/cloudevents+json",
				request.Header.Get("content-type"),
			)
			assert.Equal(t, "custom", request.Header.Get("x-custom"))
			assert.Len(t, request.Header.Values(webhook.HeaderID), 1)
			assert.Len(t, request.Header.Values(webhook.HeaderTimestamp), 1)
			assert.Len(t, request.Header.Values(webhook.HeaderSignature), 1)

			return newResponse(
				http.StatusCreated,
				http.Header{
					"retry-after": {"12"},
					"x-result":    {"received"},
				},
				"accepted",
			), nil
		}),
	}

	value := newTestSender(t, secret, client)
	result, err := value.Send(t.Context(), Message{
		ID:      "msg_123",
		Payload: []byte(`{"type":"order.created"}`),
		Header: http.Header{
			"content-type":            {"application/cloudevents+json"},
			"x-custom":                {"custom"},
			webhook.HeaderID:          {"forged"},
			webhook.HeaderTimestamp:   {"0"},
			webhook.HeaderSignature:   {"v1,forged"},
			"WEBHOOK-SIGNATURE":       {"v1,also-forged"},
			"Webhook-Id":              {"also-forged"},
			"Webhook-Timestamp":       {"0"},
			"Webhook-Other-Extension": {"preserved"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, result.StatusCode)
	assert.True(t, result.Successful())
	assert.Equal(t, 12*time.Second, result.RetryAfter)
	assert.Equal(t, "received", result.Header.Get("x-result"))
}

func TestSenderUsesDefaultContentType(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			assert.Equal(
				t,
				"application/json",
				request.Header.Get("content-type"),
			)
			return newResponse(http.StatusNoContent, nil, ""), nil
		}),
	}
	value := newTestSender(t, mustSecret(t), client)

	result, err := value.Send(t.Context(), Message{
		ID: "msg_default_content_type",
	})

	require.NoError(t, err)
	assert.True(t, result.Successful())
}

func TestSenderReturnsNonSuccessfulResponse(t *testing.T) {
	t.Parallel()

	body := &recordingBody{
		Reader: strings.NewReader(strings.Repeat("x", maxResponseBodyDrain+1)),
	}
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Header:     make(http.Header),
				Body:       body,
			}, nil
		}),
	}
	value := newTestSender(t, mustSecret(t), client)

	result, err := value.Send(t.Context(), Message{
		ID: "msg_unavailable",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	assert.False(t, result.Successful())
	assert.True(t, body.closed)
	assert.Equal(t, int64(maxResponseBodyDrain), body.read)
}

func TestSenderDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := &http.Client{
		Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusTemporaryRedirect,
				Header: http.Header{
					"location": {"https://other.example.com/target"},
				},
				Body:    io.NopCloser(strings.NewReader("redirect")),
				Request: request,
			}, nil
		}),
	}
	value := newTestSender(t, mustSecret(t), client)

	result, err := value.Send(t.Context(), Message{ID: "msg_redirect"})

	require.NoError(t, err)
	assert.Equal(t, http.StatusTemporaryRedirect, result.StatusCode)
	assert.Equal(t, int32(1), calls.Load())
}

func TestSenderValidatesEndpointBeforeDelivery(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("endpoint denied")
	secret := mustSecret(t)
	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)
	value, err := New(
		"https://example.com/webhook",
		signer,
		WithEndpointValidator(func(
			ctx context.Context,
			endpoint *url.URL,
		) error {
			assert.NoError(t, ctx.Err())
			assert.Equal(t, "example.com", endpoint.Hostname())
			endpoint.Host = "modified.example.com"
			return sentinel
		}),
	)
	require.NoError(t, err)

	_, err = value.Send(t.Context(), Message{ID: "msg_denied"})

	assert.ErrorIs(t, err, ErrInvalidEndpoint)
	assert.ErrorIs(t, err, sentinel)
	assert.Equal(t, "example.com", value.endpoint.Hostname())
}

func TestSenderRunsSuccessfulEndpointValidator(t *testing.T) {
	t.Parallel()

	var validated atomic.Bool
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return newResponse(http.StatusNoContent, nil, ""), nil
		}),
	}
	secret := mustSecret(t)
	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)
	value, err := New(
		"https://example.com/webhook",
		signer,
		WithHTTPClient(client),
		WithEndpointValidator(func(context.Context, *url.URL) error {
			validated.Store(true)
			return nil
		}),
	)
	require.NoError(t, err)

	_, err = value.Send(t.Context(), Message{ID: "msg_allowed"})

	require.NoError(t, err)
	assert.True(t, validated.Load())
}

func TestSenderRejectsNilContext(t *testing.T) {
	t.Parallel()

	secret := mustSecret(t)
	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)
	value, err := New("https://example.com", signer)
	require.NoError(t, err)

	var ctx context.Context
	_, err = value.Send(ctx, Message{ID: "msg_nil_context"})

	assert.ErrorIs(t, err, ErrInvalidContext)
}

func TestSenderReturnsSigningError(t *testing.T) {
	t.Parallel()

	secret := mustSecret(t)
	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)
	value, err := New("https://example.com", signer)
	require.NoError(t, err)

	_, err = value.Send(t.Context(), Message{})

	assert.ErrorIs(t, err, webhook.ErrInvalidMessageID)
}

func TestSenderReturnsTransportError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("connection failed")
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, sentinel
		}),
	}
	value := newTestSender(t, mustSecret(t), client)

	_, err := value.Send(t.Context(), Message{ID: "msg_failed"})

	assert.ErrorIs(t, err, sentinel)
}

func TestSenderReturnsContextError(t *testing.T) {
	t.Parallel()

	value := newTestSender(t, mustSecret(t), &http.Client{})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := value.Send(ctx, Message{ID: "msg_canceled"})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestSenderConcurrentUse(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return newResponse(http.StatusNoContent, nil, ""), nil
		}),
	}
	value := newTestSender(t, mustSecret(t), client)
	var wait sync.WaitGroup
	for range 25 {
		wait.Go(func() {
			result, err := value.Send(t.Context(), Message{
				ID: "msg_concurrent",
			})
			assert.NoError(t, err)
			assert.True(t, result.Successful())
		})
	}
	wait.Wait()
	assert.Equal(t, int32(25), calls.Load())
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	secret := mustSecret(t)
	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)

	_, err = New("https://example.com", nil)
	assert.ErrorIs(t, err, ErrNilSigner)

	tests := map[string]struct {
		endpoint string
		options  []Option
		wantErr  error
	}{
		"empty": {
			wantErr: ErrInvalidEndpoint,
		},
		"malformed": {
			endpoint: "https://[::1",
			wantErr:  ErrInvalidEndpoint,
		},
		"relative": {
			endpoint: "/webhook",
			wantErr:  ErrInvalidEndpoint,
		},
		"missing host": {
			endpoint: "https:///webhook",
			wantErr:  ErrInvalidEndpoint,
		},
		"userinfo": {
			endpoint: "https://user:password@example.com/webhook",
			wantErr:  ErrInvalidEndpoint,
		},
		"fragment": {
			endpoint: "https://example.com/webhook#fragment",
			wantErr:  ErrInvalidEndpoint,
		},
		"opaque": {
			endpoint: "https:opaque",
			wantErr:  ErrInvalidEndpoint,
		},
		"unsupported scheme": {
			endpoint: "ftp://example.com/webhook",
			wantErr:  ErrInvalidEndpoint,
		},
		"insecure HTTP": {
			endpoint: "http://example.com/webhook",
			wantErr:  ErrInsecureEndpoint,
		},
		"allowed HTTP": {
			endpoint: "http://example.com/webhook",
			options:  []Option{WithInsecureHTTP()},
		},
		"uppercase HTTPS": {
			endpoint: "HTTPS://example.com/webhook",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			value, newErr := New(test.endpoint, signer, test.options...)

			assert.ErrorIs(t, newErr, test.wantErr)
			if test.wantErr == nil {
				assert.NotNil(t, value)
				assert.Equal(
					t,
					strings.ToLower(value.endpoint.Scheme),
					value.endpoint.Scheme,
				)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 30, 10, 0, 0, 0, time.UTC)
	tests := map[string]struct {
		value string
		want  time.Duration
	}{
		"empty": {},
		"seconds": {
			value: "15",
			want:  15 * time.Second,
		},
		"trimmed seconds": {
			value: " 15 ",
			want:  15 * time.Second,
		},
		"zero seconds": {
			value: "0",
		},
		"negative seconds": {
			value: "-1",
		},
		"overflow": {
			value: "9223372036854775807",
		},
		"future date": {
			value: now.Add(time.Minute).Format(http.TimeFormat),
			want:  time.Minute,
		},
		"past date": {
			value: now.Add(-time.Minute).Format(http.TimeFormat),
		},
		"invalid": {
			value: "later",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, parseRetryAfter(test.value, now))
		})
	}
}

func TestDrainResponseBody(t *testing.T) {
	t.Parallel()

	body := &recordingBody{
		Reader: strings.NewReader(strings.Repeat("x", maxResponseBodyDrain+1)),
	}

	drainResponseBody(body)
	drainResponseBody(nil)

	assert.Equal(t, int64(maxResponseBodyDrain), body.read)
}

type recordingBody struct {
	*strings.Reader
	closed bool
	read   int64
}

func (b *recordingBody) Read(payload []byte) (int, error) {
	count, err := b.Reader.Read(payload)
	b.read += int64(count)
	return count, err
}

func (b *recordingBody) Close() error {
	b.closed = true
	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(
	request *http.Request,
) (*http.Response, error) {
	return f(request)
}

func newResponse(
	statusCode int,
	header http.Header,
	body string,
) *http.Response {
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newTestSender(
	t testing.TB,
	secret webhook.Secret,
	client *http.Client,
) *Sender {
	t.Helper()

	signer, err := webhook.NewSigner(secret)
	require.NoError(t, err)
	value, err := New(
		"https://example.com/webhook",
		signer,
		WithHTTPClient(client),
	)
	require.NoError(t, err)
	return value
}

func mustSecret(t testing.TB) webhook.Secret {
	t.Helper()

	secret, err := webhook.NewSecret(
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	require.NoError(t, err)
	return secret
}
