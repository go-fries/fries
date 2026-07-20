// Package response builds and writes a consistent JSON envelope for HTTP
// APIs.
//
// A response body contains an application status, an optional application
// code, a public message, and data. The application code is independent of the
// HTTP status code passed to [Write].
package response
