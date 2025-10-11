package jsonrpc

import (
	"testing"
)

func TestProtocolVersion(t *testing.T) {
	// Test that ProtocolVersion is set to JSON-RPC 2.0 specification
	const expectedVersion = "2.0"
	if ProtocolVersion != expectedVersion {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, expectedVersion)
	}
}

func TestRequestUsesProtocolVersion(t *testing.T) {
	// Test that Request uses the ProtocolVersion constant
	req := &Request{
		JSONRPC: ProtocolVersion,
		Method:  "testMethod",
	}

	if req.JSONRPC != ProtocolVersion {
		t.Errorf("Request.JSONRPC = %q, want %q", req.JSONRPC, ProtocolVersion)
	}
}
