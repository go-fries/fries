// Package lifecycle coordinates application startup, execution, and shutdown.
//
// Providers bootstrap in registration order and shut down in reverse order.
// Each successful provider may return a derived context for the next provider
// and, during startup, the application handler. If startup fails, providers
// that started successfully are shut down before the error is returned.
// Shutdown continues after individual provider failures and joins all lifecycle
// errors so callers can inspect every failure with errors.Is and errors.As.
package lifecycle
