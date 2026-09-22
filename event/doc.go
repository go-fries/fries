// Package event provides synchronous, type-aware dispatching for in-process
// application events.
//
// Use [Listen] to register a function with an inferred event type, or
// [Dispatcher.Subscribe] with [HandlerFor] to register handler objects.
// Both return a [Subscription] for removing the registrations.
package event
