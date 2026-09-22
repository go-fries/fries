// Package cache provides cache backend contracts and convenience operations.
//
// Typed creates a reusable view with typed Get, Set and Remember operations
// over a ReadWriter. Its Remember loader receives the operation's Context,
// and its writes report ErrWriteUnconfirmed for a negative backend status.
//
// NewRepository wraps a Store with aliases and conditional writes. Its Add
// method can fall back to a non-atomic check followed by a write. Atomicity
// depends on the backend's Add implementation.
package cache
