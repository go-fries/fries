package jsonrpc //nolint:revive

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
