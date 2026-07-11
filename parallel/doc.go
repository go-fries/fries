// Package parallel provides context-aware helpers for concurrent batch work.
//
// It supports unbounded and bounded task execution, concurrent iteration, and
// order-preserving concurrent mapping and filtering. Fail-fast helpers cancel
// sibling work through context.Context, while MapResults supports best-effort
// processing with one outcome per input value.
package parallel
