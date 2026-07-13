// Package poll provides context-aware condition polling.
//
// Polling begins with an immediate condition check. When the condition is not
// yet satisfied, the package waits for a fixed interval before checking again.
// The caller's context owns the total polling lifetime, including interval
// waits. Conditions run synchronously and should observe the supplied context
// while performing blocking work.
//
// Package poll is intended for observing state until a condition is satisfied.
// Transiently failing operations that should be executed again belong in a
// retry workflow instead.
package poll
