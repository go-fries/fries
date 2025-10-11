package jsonrpc

import (
	"testing"
)

const expectedProtocolVersion = "2.0"

func TestProtocolVersion(t *testing.T) {
	// Test that ProtocolVersion is set to JSON-RPC 2.0 specification
	if ProtocolVersion != expectedProtocolVersion {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, expectedProtocolVersion)
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
