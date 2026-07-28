// Package memory provides an in-process idempotency store.
//
// Records are lost when the process exits. [Store] is safe for concurrent use,
// expires records lazily, and does not start background goroutines.
package memory
