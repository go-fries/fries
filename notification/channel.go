package notification

import (
	"context"
	"time"
)

// Channel performs one delivery attempt. Nil means the channel accepted the
// delivery, not necessarily that a person received or read it. Implementations
// must honor ctx and be safe for concurrent calls when shared by a Manager.
type Channel interface {
	Send(ctx context.Context, delivery Delivery) error
}

// ChannelFunc adapts a function to [Channel].
type ChannelFunc func(context.Context, Delivery) error

// Send calls f(ctx, delivery).
func (f ChannelFunc) Send(ctx context.Context, delivery Delivery) error {
	return f(ctx, delivery)
}

// Delivery describes one recipient/channel delivery. ID remains stable when
// the same notification ID, recipient key and channel are used again. A stable
// ID is not itself a deduplication or exactly-once guarantee.
type Delivery struct {
	ID             string
	NotificationID string
	Channel        string
	Recipient      Recipient
	Notification   Notification
	CreatedAt      time.Time
}
