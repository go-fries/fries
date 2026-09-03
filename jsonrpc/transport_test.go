package jsonrpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNewHTTPTransport(t *testing.T) {
	customClient := &http.Client{}
	transport := NewHTTPTransport(
		"https://example.com/",
		WithHTTPTransportUserAgent("fries-test"),
		WithHTTPTransportClient(customClient),
	)

	assert.Equal(t, "https://example.com", transport.addr)
	assert.Equal(t, "fries-test", transport.userAgent)
	assert.Same(t, customClient, transport.httpClient)

	defaultTransport := NewHTTPTransport("https://example.com")
	assert.Same(t, http.DefaultClient, defaultTransport.httpClient)
}

func TestHTTPTransportSend(t *testing.T) {
	requests := make(chan *http.Request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Clone(request.Context())

		var rpcRequest Request
		if err := json.NewDecoder(request.Body).Decode(&rpcRequest); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		assert.Equal(t, "method", rpcRequest.Method)

		writer.Header().Set("Content-Type", "application/json")
		_, err := io.WriteString(writer, `{"jsonrpc":"2.0","result":"ok","id":"request-id"}`)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)

	transport := NewHTTPTransport(
		server.URL+"/",
		WithHTTPTransportClient(server.Client()),
		WithHTTPTransportUserAgent("fries-test"),
	)
	request := &Request{
		JSONRPC: ProtocolVersion,
		Method:  "method",
		Params:  json.RawMessage(`[]`),
		ID:      NewID("request-id"),
	}

	resp, err := transport.Send(t.Context(), "", request)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.JSONEq(t, `"ok"`, string(resp.Result))
	received := <-requests
	assert.Equal(t, "/", received.URL.Path)
	assert.Equal(t, http.MethodPost, received.Method)
	assert.Equal(t, "application/json", received.Header.Get("Content-Type"))
	assert.Equal(t, "fries-test", received.Header.Get("User-Agent"))

	_, err = transport.Send(t.Context(), "/users", request)
	require.NoError(t, err)
	received = <-requests
	assert.Equal(t, "/users", received.URL.Path)
}

func TestHTTPTransportSendErrors(t *testing.T) {
	sentinel := errors.New("sentinel")

	t.Run("encode request", func(t *testing.T) {
		transport := NewHTTPTransport("https://example.com")
		resp, err := transport.Send(t.Context(), "", &Request{Params: json.RawMessage(`{`)})

		assert.Nil(t, resp)
		require.Error(t, err)
	})

	t.Run("create request", func(t *testing.T) {
		transport := NewHTTPTransport("://invalid")
		resp, err := transport.Send(t.Context(), "", &Request{})

		assert.Nil(t, resp)
		require.Error(t, err)
	})

	t.Run("send request", func(t *testing.T) {
		transport := NewHTTPTransport("https://example.com", WithHTTPTransportClient(&http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, sentinel
			}),
		}))

		resp, err := transport.Send(t.Context(), "", &Request{})

		assert.Nil(t, resp)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("decode response", func(t *testing.T) {
		transport := NewHTTPTransport("https://example.com", WithHTTPTransportClient(&http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("not-json")),
				}, nil
			}),
		}))

		resp, err := transport.Send(t.Context(), "", &Request{})

		assert.Nil(t, resp)
		require.Error(t, err)
	})
}
