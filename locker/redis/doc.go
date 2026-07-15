// Package redis implements distributed leases backed by Redis.
//
// A Redis lease is valid only until its TTL expires. Applications that require
// stronger correctness should also enforce invariants at the protected
// resource, for example with idempotency or database constraints.
package redis
