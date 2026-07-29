// Package redis provides a Redis-backed rate-limit store.
//
// The store uses a single-key Lua script and Redis server time to share
// capacity across application processes.
package redis
