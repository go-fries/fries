// Package ratelimit provides context-aware, key-based rate limiting.
//
// A [Limiter] binds one [Limit] to a [Store]. Calls return an immediate
// [Decision]; they never wait for capacity or treat a rejected request as an
// error.
package ratelimit
