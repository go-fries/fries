// Package cache provides cache backend contracts and convenience operations.
//
// NewRepository wraps a Store with aliases and conditional writes. Its Add
// method can fall back to a non-atomic check followed by a write. Atomicity
// depends on the backend's Add implementation.
package cache
