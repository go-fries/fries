// Package idempotency coordinates idempotent business operations through an
// atomic Store.
//
// An Executor claims a key before running a Handler. Concurrent callers for
// the same key receive ErrInProgress, while calls made after successful
// completion return without running the Handler again. Optional fingerprints
// detect reuse of a key for different input.
//
// The package does not provide exactly-once execution. Applications should
// still use database constraints, transactions, or downstream idempotency
// mechanisms for correctness-critical side effects.
package idempotency
