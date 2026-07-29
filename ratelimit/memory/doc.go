// Package memory provides an in-process rate-limit store.
//
// The store is intended for single-process services, local development, and
// tests. Separate processes do not share capacity.
package memory
