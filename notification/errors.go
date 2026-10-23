package notification

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidConfig indicates an invalid or duplicate channel registration.
	ErrInvalidConfig = errors.New("notification: invalid configuration")
	// ErrInvalidNotification indicates a nil notification or empty business type.
	ErrInvalidNotification = errors.New("notification: invalid notification")
	// ErrInvalidRecipient indicates a nil recipient or invalid persistent key.
	ErrInvalidRecipient = errors.New("notification: invalid recipient")
	// ErrInvalidID indicates an empty explicitly supplied notification ID.
	ErrInvalidID = errors.New("notification: invalid ID")
	// ErrUnknownChannel indicates a selected channel was not registered.
	ErrUnknownChannel = errors.New("notification: unknown channel")
	// ErrUnsupportedNotification indicates a missing channel message capability.
	ErrUnsupportedNotification = errors.New("notification: unsupported notification")
	// ErrMissingRoute indicates a missing channel-specific recipient route.
	ErrMissingRoute = errors.New("notification: missing route")
)

// DeliveryError associates a delivery failure with its channel and stable ID.
// Errors returned by Send can be inspected with errors.Is and errors.As.
type DeliveryError struct {
	ID      string
	Channel string
	Err     error
}

// Error describes the channel, delivery ID and original failure.
func (e *DeliveryError) Error() string {
	return fmt.Sprintf("notification: channel %q delivery %q: %v", e.Channel, e.ID, e.Err)
}

// Unwrap returns the original failure.
func (e *DeliveryError) Unwrap() error {
	return e.Err
}
