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
//
// Workers expose the current delivery attempt through Task.Attempt. Queue
// implementations increment the attempt before handler execution, so the first
// handler invocation sees Attempt equal to 1.
//
// Handlers normally return an error to let the Worker's RetryPolicy decide the
// next action. For explicit business decisions, handlers may return ErrDiscard,
// RetryAfter, or DeadLetter.
package queue
