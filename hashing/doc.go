// Package hashing computes deterministic digests from byte slices, strings,
// streams, and files.
//
// Hashers use the standard library's hash.Hash interface and are safe to reuse
// across concurrent operations when their NewHash function is concurrency-safe.
// The package is intended for checksums and content fingerprints. Password
// hashing and message authentication require dedicated APIs with different
// security contracts.
package hashing
