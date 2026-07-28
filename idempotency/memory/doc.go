// Package memory provides an in-process Store for idempotent operations.
//
// Records are lost when the process exits. The Store is concurrency-safe,
// expires records lazily, and does not start background goroutines.
package memory
