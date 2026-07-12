// Package retry adapts the base retry component to Hyperf Jet middleware.
//
// The middleware retries timeout errors, HTTP 408 and 429 responses, and HTTP
// 5xx responses by default. Callers configure attempts, backoff, predicates,
// and notifications with github.com/go-fries/fries/retry/v4 options.
package retry
