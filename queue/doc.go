// Package queue provides durable task queue primitives.
//
// The package defines producer, consumer, worker, retry, and middleware
// contracts that can be backed by concrete queue implementations. Delivery is
// at least once: handlers should be idempotent because a task may be delivered
// again after a process crash, timeout, or retryable error.
//
// Consumers use a blocking receive model. Consumer.Receive returns a delivery,
// the receive context error, ErrConsumerClosed when the consumer is closed, or a
// backend error. Malformed backend messages are returned as backend errors so
// adapters can make their own acknowledgement or rejection decision.
package queue
