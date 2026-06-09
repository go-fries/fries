// Package memory provides an in-memory queue implementation for tests,
// examples, and local development.
//
// It is not a production adapter. Tasks are not persisted across process
// restarts, and in-flight tasks are not recovered if the process exits before a
// handler retries or dead-letters them.
package memory
