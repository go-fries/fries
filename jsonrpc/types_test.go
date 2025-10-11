package jsonrpc

import (
	"testing"
)

func TestProtocolVersion(t *testing.T) {
	// Test that ProtocolVersion is set to JSON-RPC 2.0 specification
	expected := "2.0"
	if ProtocolVersion != expected {
		t.Errorf("ProtocolVersion = %q, want %q", ProtocolVersion, expected)
	}
}

func TestRequestUsesProtocolVersion(t *testing.T) {
	// Test that Request uses the ProtocolVersion constant
	req := &Request{
		JSONRPC: ProtocolVersion,
		Method:  "testMethod",
	}

	expected := "2.0"
	if req.JSONRPC != expected {
		t.Errorf("Request.JSONRPC = %q, want %q", req.JSONRPC, expected)
	}
}
