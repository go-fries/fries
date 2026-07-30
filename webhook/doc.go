// Package webhook signs and verifies HTTP webhook messages using the
// Standard Webhooks HMAC-SHA256 scheme.
//
// A [Signer] protects the message ID, timestamp, and exact payload bytes. A
// [Verifier] authenticates those values and enforces a timestamp tolerance.
// The package does not send requests, retry deliveries, decode payloads, or
// prevent duplicate business execution.
package webhook
