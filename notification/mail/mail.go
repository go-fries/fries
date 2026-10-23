package mail

import (
	"context"
	"errors"
	"fmt"
	netmail "net/mail"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/internal/nilvalue"
)

// Name is the conventional mail channel name.
const Name = "mail"

// ErrInvalidAddress indicates a malformed mailbox address.
var ErrInvalidAddress = errors.New("notification/mail: invalid address")

// Notification is the message-building capability required by this channel.
type Notification interface {
	ToMail(ctx context.Context, to notification.Recipient) (Message, error)
}

// RouteProvider supplies a recipient's mailbox. The Address field must contain
// one bare mailbox address; the Name field may contain a display name.
type RouteProvider interface {
	MailRoute(ctx context.Context) (netmail.Address, error)
}

// Message is the mail representation of a notification. The sender owns From,
// transport settings, MIME construction and any application-specific defaults.
type Message struct {
	Subject  string
	TextBody string
	HTMLBody string
}

// Envelope supplies a sender with the destination, content and delivery IDs.
// ID is stable across retries and can be passed to a provider's idempotency API.
type Envelope struct {
	ID             string
	NotificationID string
	To             netmail.Address
	Message        Message
}

// Sender sends one mail envelope. Implementations should honor ctx and should
// not hide retry loops. Success normally means provider acceptance, not receipt.
type Sender interface {
	Send(ctx context.Context, envelope Envelope) error
}

// SenderFunc adapts a function to [Sender].
type SenderFunc func(context.Context, Envelope) error

// Send calls f(ctx, envelope).
func (f SenderFunc) Send(ctx context.Context, envelope Envelope) error {
	return f(ctx, envelope)
}

// Channel builds and sends a mail representation using an injected Sender.
type Channel struct {
	sender Sender
}

// NewChannel creates a mail channel. A nil sender, including a typed nil,
// returns notification.ErrInvalidConfig. The caller owns the sender's resources.
func NewChannel(sender Sender) (*Channel, error) {
	if nilvalue.IsNil(sender) {
		return nil, fmt.Errorf("%w: mail sender is nil", notification.ErrInvalidConfig)
	}
	return &Channel{sender: sender}, nil
}

// Send resolves a mailbox, builds the message, and calls the sender once.
// Missing capabilities or routes and invalid addresses are returned as errors.
func (c *Channel) Send(ctx context.Context, d notification.Delivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	builder, ok := d.Notification.(Notification)
	if !ok || nilvalue.IsNil(builder) {
		return fmt.Errorf("%w: mail needs ToMail", notification.ErrUnsupportedNotification)
	}
	router, ok := d.Recipient.(RouteProvider)
	if !ok || nilvalue.IsNil(router) {
		return fmt.Errorf("%w: mail needs MailRoute", notification.ErrMissingRoute)
	}
	address, err := router.MailRoute(ctx)
	if err != nil {
		return err
	}
	if err := validateAddress(address); err != nil {
		return err
	}
	message, err := builder.ToMail(ctx, d.Recipient)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.sender.Send(ctx, Envelope{
		ID:             d.ID,
		NotificationID: d.NotificationID,
		To:             address,
		Message:        message,
	})
}

// Recipient is an anonymous, mail-only recipient created by To. It has no
// persistent inbox identity. Reuse the same logical notification ID only for
// the same anonymous recipient and notification data.
type Recipient struct {
	address netmail.Address
}

// To parses a mailbox, optionally including a display name, for on-demand mail.
func To(address string) (Recipient, error) {
	parsed, err := netmail.ParseAddress(address)
	if err != nil {
		return Recipient{}, fmt.Errorf("%w: %w", ErrInvalidAddress, err)
	}
	return Recipient{address: *parsed}, nil
}

// NotificationKey returns a zero key because this recipient has no inbox.
func (Recipient) NotificationKey() notification.RecipientKey {
	return notification.RecipientKey{}
}

// MailRoute returns the mailbox parsed by To.
func (r Recipient) MailRoute(ctx context.Context) (netmail.Address, error) {
	if err := ctx.Err(); err != nil {
		return netmail.Address{}, err
	}
	return r.address, nil
}

func validateAddress(address netmail.Address) error {
	if address.Address == "" {
		return notification.ErrMissingRoute
	}
	parsed, err := netmail.ParseAddress(address.Address)
	if err != nil || parsed.Address != address.Address {
		return ErrInvalidAddress
	}
	return nil
}

var (
	_ notification.Channel   = (*Channel)(nil)
	_ notification.Recipient = Recipient{}
	_ RouteProvider          = Recipient{}
)
