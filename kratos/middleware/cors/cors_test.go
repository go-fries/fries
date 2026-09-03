package cors

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kratos/kratos/v3/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type headerCarrier http.Header

func (h headerCarrier) Get(key string) string {
	return http.Header(h).Get(key)
}

func (h headerCarrier) Set(key, value string) {
	http.Header(h).Set(key, value)
}

func (h headerCarrier) Add(key, value string) {
	http.Header(h).Add(key, value)
}

func (h headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for key := range h {
		keys = append(keys, key)
	}
	return keys
}

func (h headerCarrier) Values(key string) []string {
	return http.Header(h).Values(key)
}

type mockTransport struct {
	kind        transport.Kind
	requestHead headerCarrier
	replyHead   headerCarrier
}

func newMockTransport(kind transport.Kind) *mockTransport {
	return &mockTransport{
		kind:        kind,
		requestHead: headerCarrier{},
		replyHead:   headerCarrier{},
	}
}

func (m *mockTransport) Kind() transport.Kind {
	return m.kind
}

func (m *mockTransport) Endpoint() string {
	return ""
}

func (m *mockTransport) Operation() string {
	return ""
}

func (m *mockTransport) RequestHeader() transport.Header {
	return m.requestHead
}

func (m *mockTransport) ReplyHeader() transport.Header {
	return m.replyHead
}

type mockHTTPTransport struct {
	*mockTransport
	request *http.Request
}

func (m *mockHTTPTransport) Request() *http.Request {
	return m.request
}

func (m *mockHTTPTransport) PathTemplate() string {
	return ""
}

func TestOptions(t *testing.T) {
	op := newOptions()
	assert.Equal(t, []string{"*"}, op.paths)
	assert.Equal(t, []string{"*"}, op.allowedMethods)
	assert.Equal(t, []string{"*"}, op.allowedHeaders)
	assert.Equal(t, []string{"*"}, op.allowedOrigins)

	options := []Option{
		Path("/api/*"),
		AppendPath("/health"),
		AllowedMethods(http.MethodGet),
		AppendAllowedMethods(http.MethodPost),
		AllowedHeaders("Content-Type"),
		AppendAllowedHeaders("Authorization"),
		AllowedOrigins("https://example.com"),
		AppendAllowedOrigins("https://api.example.com"),
	}
	for _, option := range options {
		option(op)
	}

	assert.Equal(t, []string{"/api/*", "/health"}, op.paths)
	assert.Equal(t, []string{http.MethodGet, http.MethodPost}, op.allowedMethods)
	assert.Equal(t, []string{"Content-Type", "Authorization"}, op.allowedHeaders)
	assert.Equal(t, []string{"https://example.com", "https://api.example.com"}, op.allowedOrigins)
}

func TestIsPath(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		{name: "all paths", pattern: "*", value: "/api/users", want: true},
		{name: "exact", pattern: "/api/users", value: "/api/users", want: true},
		{name: "leading slash", pattern: "api/users", value: "/api/users", want: true},
		{name: "wildcard", pattern: "/api/*", value: "/api/users/1", want: true},
		{name: "mismatch", pattern: "/api/*", value: "/admin/users", want: false},
		{name: "invalid expression", pattern: "[", value: "/api/users", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPath(tt.pattern, tt.value))
		})
	}
}

func TestCorsPassesThroughWithoutHTTPRequest(t *testing.T) {
	sentinel := errors.New("handler error")
	request := &struct{}{}
	wantReply := &struct{}{}
	tests := []struct {
		name string
		ctx  func() context.Context
	}{
		{
			name: "missing transport",
			ctx:  t.Context,
		},
		{
			name: "non HTTP transport",
			ctx: func() context.Context {
				return transport.NewServerContext(t.Context(), newMockTransport(transport.KindGRPC))
			},
		},
		{
			name: "missing HTTP request",
			ctx: func() context.Context {
				return transport.NewServerContext(t.Context(), newMockTransport(transport.KindHTTP))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := func(_ context.Context, gotRequest any) (any, error) {
				called = true
				assert.Same(t, request, gotRequest)
				return wantReply, sentinel
			}

			reply, err := Cors()(handler)(tt.ctx(), request)

			assert.True(t, called)
			assert.Same(t, wantReply, reply)
			assert.ErrorIs(t, err, sentinel)
		})
	}
}

func TestCorsAddsHeadersForMatchingPath(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodOptions, "https://service.example.com/api/users", nil)
	tr := &mockHTTPTransport{
		mockTransport: newMockTransport(transport.KindHTTP),
		request:       httpRequest,
	}
	ctx := transport.NewServerContext(t.Context(), tr)
	request := &struct{}{}
	handler := func(_ context.Context, gotRequest any) (any, error) {
		assert.Same(t, request, gotRequest)
		return gotRequest, nil
	}
	middleware := Cors(
		Path("/api/*", "*"),
		AllowedMethods(http.MethodGet, http.MethodPost),
		AllowedHeaders("Content-Type", "Authorization"),
		AllowedOrigins("https://client.example.com"),
	)

	reply, err := middleware(handler)(ctx, request)

	require.NoError(t, err)
	assert.Same(t, request, reply)
	assert.Equal(t, []string{"true"}, tr.replyHead.Values("Access-Control-Allow-Credentials"))
	assert.Equal(t, []string{"GET, POST"}, tr.replyHead.Values("Access-Control-Allow-Methods"))
	assert.Equal(t, []string{"Content-Type, Authorization"}, tr.replyHead.Values("Access-Control-Allow-Headers"))
	assert.Equal(t, []string{"https://client.example.com"}, tr.replyHead.Values("Access-Control-Allow-Origin"))
}

func TestCorsUsesDefaults(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodGet, "https://service.example.com/health", nil)
	tr := &mockHTTPTransport{
		mockTransport: newMockTransport(transport.KindHTTP),
		request:       httpRequest,
	}
	ctx := transport.NewServerContext(t.Context(), tr)
	handler := func(_ context.Context, request any) (any, error) {
		return request, nil
	}

	_, err := Cors()(handler)(ctx, nil)

	require.NoError(t, err)
	assert.Equal(t, "*", tr.replyHead.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "*", tr.replyHead.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "*", tr.replyHead.Get("Access-Control-Allow-Origin"))
}

func TestCorsDoesNotAddHeadersForUnmatchedPath(t *testing.T) {
	httpRequest := httptest.NewRequest(http.MethodGet, "https://service.example.com/admin/users", nil)
	tr := &mockHTTPTransport{
		mockTransport: newMockTransport(transport.KindHTTP),
		request:       httpRequest,
	}
	ctx := transport.NewServerContext(t.Context(), tr)
	handlerCalled := false
	handler := func(_ context.Context, request any) (any, error) {
		handlerCalled = true
		return request, nil
	}

	_, err := Cors(Path("/api/*"))(handler)(ctx, nil)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Empty(t, tr.replyHead)
}
