package notification

import "context"

// Notification describes a business notification. Type is a stable business
// identifier, such as "invoice.paid.v1", rather than a Go type name.
// Via selects channel names in execution order. Duplicate names are sent once.
// Notifications must not be mutated while a Send is in progress.
type Notification interface {
	Type() string
	Via(ctx context.Context, to Recipient) ([]string, error)
}

// SendGuard is an optional capability of a [Notification]. It is checked before
// each channel's Send. False with no error skips that delivery; an error fails
// that delivery without preventing other channels from being attempted.
type SendGuard interface {
	ShouldSend(ctx context.Context, to Recipient, channel string) (bool, error)
}
