// Package queue provides durable task queue primitives.
//
// The package defines producer, consumer, worker, retry, and middleware
// contracts that can be backed by concrete queue implementations. Delivery is
// at least once: handlers should be idempotent because a task may be delivered
// again after a process crash, timeout, or retryable error.
package queue
