// Package health runs named application health checks and exposes their
// results as structured reports or standard HTTP probe responses.
//
// Applications should generally use separate registries for liveness and
// readiness. Checks run with a shared time budget and bounded concurrency.
// Health does not cache results, run background checks, or report failures to
// logging and alerting systems.
package health
