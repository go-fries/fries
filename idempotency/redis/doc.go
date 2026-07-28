// Package redis provides a Redis-backed Store for idempotent operations.
//
// State transitions use Lua scripts so claims and completed records can be
// shared safely across application processes.
package redis
