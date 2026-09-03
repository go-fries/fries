package jsonrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolVersion(t *testing.T) {
	// Test that ProtocolVersion is set to JSON-RPC 2.0 specification
	assert.Equal(t, "2.0", ProtocolVersion)
}

func TestRequestUsesProtocolVersion(t *testing.T) {
	// Test that Request uses the ProtocolVersion constant
	req := &Request{
		JSONRPC: ProtocolVersion,
		Method:  "testMethod",
	}

	assert.Equal(t, ProtocolVersion, req.JSONRPC)
}

func TestError(t *testing.T) {
	err := &Error{Code: -32601, Message: "method not found"}

	assert.Equal(t, "code: -32601, message: method not found", err.Error())
	unwrapped := err.Unwrap()
	require.Error(t, unwrapped)
	assert.Equal(t, "method not found", unwrapped.Error())
}
