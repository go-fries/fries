package jet

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type marshalError struct {
	err error
}

func (e marshalError) Error() string {
	return e.err.Error()
}

func (e marshalError) MarshalJSON() ([]byte, error) {
	return nil, e.err
}

type structuredError struct {
	Reason    string `json:"reason"`
	Retryable bool   `json:"retryable"`
}

func (e structuredError) Error() string {
	return e.Reason
}

func TestRPCResponseError(t *testing.T) {
	sentinel := errors.New("upstream error")
	err := &RPCResponseError{
		ID:      "request-id",
		Code:    -32603,
		Message: "internal error",
		Err:     sentinel,
	}

	assert.Equal(t, "code: -32603, message: internal error, error: upstream error", err.Error())
}

func TestJSONRPCFormatterKind(t *testing.T) {
	formatter := NewJSONRPCFormatter()

	assert.Equal(t, FormatterKindJSONRPC, formatter.Kind())
	assert.Equal(t, FormatterKindJSONRPC, DefaultFormatter.Kind())
}

func TestJSONRPCFormatterRequest(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	want := &RPCRequest{
		ID:     "request-id",
		Path:   "/users/find",
		Params: json.RawMessage(`{"id":123}`),
	}

	data, err := formatter.FormatRequest(want)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"jsonrpc": "2.0",
		"method": "/users/find",
		"params": {"id": 123},
		"id": "request-id"
	}`, string(data))

	got, err := formatter.ParseRequest(data)
	require.NoError(t, err)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Path, got.Path)
	assert.JSONEq(t, string(want.Params), string(got.Params))
}

func TestJSONRPCFormatterRequestErrors(t *testing.T) {
	formatter := NewJSONRPCFormatter()

	data, err := formatter.FormatRequest(&RPCRequest{Params: json.RawMessage(`{`)})
	assert.Nil(t, data)
	require.Error(t, err)

	request, err := formatter.ParseRequest([]byte("not-json"))
	assert.Nil(t, request)
	require.Error(t, err)
}

func TestJSONRPCFormatterSuccessResponse(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	want := &RPCResponse{
		ID:     "request-id",
		Result: json.RawMessage(`{"name":"Alice"}`),
	}

	data, err := formatter.FormatResponse(want, nil)
	require.NoError(t, err)

	got, err := formatter.ParseResponse(data)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
	assert.JSONEq(t, string(want.Result), string(got.Result))
}

func TestJSONRPCFormatterErrorResponse(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	want := &RPCResponseError{
		ID:      "request-id",
		Code:    -32601,
		Message: "method not found",
		Err:     errors.New("upstream error"),
	}

	data, err := formatter.FormatResponse(nil, want)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"jsonrpc": "2.0",
		"result": null,
		"id": "request-id",
		"error": {
			"code": -32601,
			"message": "method not found",
			"data": "upstream error"
		}
	}`, string(data))

	response, err := formatter.ParseResponse(data)
	assert.Nil(t, response)
	var got *RPCResponseError
	require.ErrorAs(t, err, &got)
	assert.Equal(t, want.ID, got.ID)
	assert.Equal(t, want.Code, got.Code)
	assert.Equal(t, want.Message, got.Message)
	require.Error(t, got.Err)
	assert.Equal(t, "upstream error", got.Err.Error())
}

func TestJSONRPCFormatterStructuredErrorData(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	data := []byte(`{
		"jsonrpc": "2.0",
		"id": "request-id",
		"error": {
			"code": -32603,
			"message": "internal error",
			"data": {"retryable": true}
		}
	}`)

	response, err := formatter.ParseResponse(data)
	assert.Nil(t, response)
	var rpcErr *RPCResponseError
	require.ErrorAs(t, err, &rpcErr)
	require.Error(t, rpcErr.Err)
	assert.JSONEq(t, `{"retryable":true}`, rpcErr.Err.Error())

	encoded, err := formatter.FormatResponse(nil, rpcErr)
	require.NoError(t, err)
	var envelope JSONRPCFormatterResponse
	require.NoError(t, json.Unmarshal(encoded, &envelope))
	require.NotNil(t, envelope.Error)
	require.Error(t, envelope.Error.Data)
	assert.JSONEq(t, `{"retryable":true}`, envelope.Error.Data.Error())
}

func TestJSONRPCFormatterStructuredError(t *testing.T) {
	formatter := NewJSONRPCFormatter()

	data, err := formatter.FormatResponse(nil, &RPCResponseError{
		ID:      "request-id",
		Code:    -32603,
		Message: "internal error",
		Err: structuredError{
			Reason:    "busy",
			Retryable: true,
		},
	})
	require.NoError(t, err)

	var envelope JSONRPCFormatterResponse
	require.NoError(t, json.Unmarshal(data, &envelope))
	require.NotNil(t, envelope.Error)
	require.Error(t, envelope.Error.Data)
	assert.JSONEq(t, `{"reason":"busy","retryable":true}`, envelope.Error.Data.Error())
}

func TestJSONRPCFormatterEmptyErrorData(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	formatted, err := formatter.FormatResponse(nil, &RPCResponseError{
		ID:      "request-id",
		Code:    -32601,
		Message: "method not found",
	})
	require.NoError(t, err)
	assert.NotContains(t, string(formatted), `"data"`)

	for _, data := range []string{
		`{"jsonrpc":"2.0","id":"request-id","error":{"code":-32601,"message":"method not found"}}`,
		`{"jsonrpc":"2.0","id":"request-id","error":{"code":-32601,"message":"method not found","data":null}}`,
	} {
		response, err := formatter.ParseResponse([]byte(data))
		assert.Nil(t, response)
		var rpcErr *RPCResponseError
		require.ErrorAs(t, err, &rpcErr)
		assert.NoError(t, rpcErr.Err)
	}
}

func TestJSONRPCFormatterResponseErrors(t *testing.T) {
	formatter := NewJSONRPCFormatter()
	sentinel := errors.New("marshal response")

	data, err := formatter.FormatResponse(nil, &RPCResponseError{Err: marshalError{err: sentinel}})
	assert.Nil(t, data)
	assert.ErrorIs(t, err, sentinel)

	response, err := formatter.ParseResponse([]byte("not-json"))
	assert.Nil(t, response)
	require.Error(t, err)

	var responseErr JSONRPCFormatterResponseError
	require.Error(t, responseErr.UnmarshalJSON([]byte("not-json")))
}
