// Package coroutines provides context-aware helpers for concurrent batch work.
//
// It supports unbounded and bounded task execution, concurrent iteration, and
// order-preserving concurrent mapping. All operations wait for started work,
// propagate the first error, and cancel sibling work through context.Context.
package coroutines
