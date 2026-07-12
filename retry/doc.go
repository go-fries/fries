// Package retry executes transiently failing operations with bounded,
// context-aware retries.
//
// Retry delays are controlled by reusable Backoff functions. Callers can stop
// retrying with Permanent, override the next delay with After, and restrict
// retryable errors with WithRetryIf. The caller's context owns the total retry
// lifetime, including waits between attempts. Contexts must not be nil, and a
// nil operation panics. Invalid optional values are ignored or normalized when
// execution can remain well-defined, while backoff constructors reject invalid
// durations immediately.
package retry
