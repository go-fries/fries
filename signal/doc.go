// Package signal routes operating system signals to functions and handler
// objects. Register handlers with Router.Listen or Router.Register, then run
// the Router with Router.Run. Router.Shutdown stops new work and waits for active
// handlers with a caller-supplied deadline.
//
// Each registration executes serially with one pending notification slot.
// Different registrations may execute concurrently. Full slots drop new
// notifications and log a warning; OS-level signal delivery can also lose or
// coalesce notifications. This package is not a reliable task queue.
//
// The package subscribes only to explicitly registered signals. It does not
// manage application shutdown ordering, exit the process, or install a default
// SIGINT/SIGTERM policy. Use os/signal.NotifyContext for applications that only
// need to cancel a context when an exit signal arrives.
package signal
