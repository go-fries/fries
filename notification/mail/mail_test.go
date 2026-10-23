package mail_test

import (
	"context"
	"errors"
	netmail "net/mail"
	"testing"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type basicNotice struct{}

func (basicNotice) Type() string { return "invoice.paid.v1" }
func (basicNotice) Via(context.Context, notification.Recipient) ([]string, error) {
	return []string{mail.Name}, nil
}

type mailNotice struct {
	basicNotice
	build func(context.Context, notification.Recipient) (mail.Message, error)
}

func (n mailNotice) ToMail(ctx context.Context, to notification.Recipient) (mail.Message, error) {
	if n.build != nil {
		return n.build(ctx, to)
	}
	return mail.Message{Subject: "Invoice paid", TextBody: "Thank you.", HTMLBody: "<p>Thank you.</p>"}, nil
}

type routedRecipient struct {
	route func(context.Context) (netmail.Address, error)
}

func (r routedRecipient) NotificationKey() notification.RecipientKey {
	return notification.RecipientKey{Type: "user", ID: "42"}
}

func (r routedRecipient) MailRoute(ctx context.Context) (netmail.Address, error) {
	return r.route(ctx)
}

type unroutedRecipient struct{}

func (unroutedRecipient) NotificationKey() notification.RecipientKey {
	return notification.RecipientKey{}
}

func TestMailChannelSendsOneEnvelope(t *testing.T) {
	var envelopes []mail.Envelope
	channel, err := mail.NewChannel(mail.SenderFunc(func(_ context.Context, envelope mail.Envelope) error {
		envelopes = append(envelopes, envelope)
		return nil
	}))
	require.NoError(t, err)
	manager, err := notification.New(notification.WithChannel(mail.Name, channel))
	require.NoError(t, err)
	to, err := mail.To("Taylor <taylor@example.com>")
	require.NoError(t, err)
	result, err := manager.Send(t.Context(), to, mailNotice{}, notification.WithID("invoice-42"))
	require.NoError(t, err)
	require.Len(t, envelopes, 1)
	assert.Equal(t, "taylor@example.com", envelopes[0].To.Address)
	assert.Equal(t, "Taylor", envelopes[0].To.Name)
	assert.Equal(t, "Invoice paid", envelopes[0].Message.Subject)
	assert.Equal(t, "Thank you.", envelopes[0].Message.TextBody)
	assert.Equal(t, "<p>Thank you.</p>", envelopes[0].Message.HTMLBody)
	assert.Equal(t, result.Deliveries[0].ID, envelopes[0].ID)
	assert.Equal(t, result.NotificationID, envelopes[0].NotificationID)
	assert.Zero(t, to.NotificationKey())
}

func TestMailChannelRejectsMissingCapabilitiesAndBadRoutes(t *testing.T) {
	routeErr := errors.New("route lookup failed")
	valid, err := mail.To("user@example.com")
	require.NoError(t, err)
	var nilNotice *mailNotice
	var nilRoute *routedRecipient
	for _, tc := range []struct {
		name string
		n    notification.Notification
		to   notification.Recipient
		want error
	}{
		{"unsupported", basicNotice{}, valid, notification.ErrUnsupportedNotification},
		{"typed nil builder", nilNotice, valid, notification.ErrUnsupportedNotification},
		{"missing route", mailNotice{}, unroutedRecipient{}, notification.ErrMissingRoute},
		{"typed nil route", mailNotice{}, nilRoute, notification.ErrMissingRoute},
		{"empty route", mailNotice{}, mail.Recipient{}, notification.ErrMissingRoute},
		{"bad route", mailNotice{}, routedRecipient{route: func(context.Context) (netmail.Address, error) {
			return netmail.Address{Address: "not-an-address"}, nil
		}}, mail.ErrInvalidAddress},
		{"display name in bare address", mailNotice{}, routedRecipient{route: func(context.Context) (netmail.Address, error) {
			return netmail.Address{Address: "User <user@example.com>"}, nil
		}}, mail.ErrInvalidAddress},
		{"route failure", mailNotice{}, routedRecipient{route: func(context.Context) (netmail.Address, error) {
			return netmail.Address{}, routeErr
		}}, routeErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			channel, err := mail.NewChannel(mail.SenderFunc(func(context.Context, mail.Envelope) error {
				calls++
				return nil
			}))
			require.NoError(t, err)
			err = channel.Send(t.Context(), notification.Delivery{Recipient: tc.to, Notification: tc.n})
			require.ErrorIs(t, err, tc.want)
			assert.Zero(t, calls)
		})
	}
}

func TestMailChannelPreservesBuildAndSenderErrors(t *testing.T) {
	want := errors.New("failed")
	to, err := mail.To("user@example.com")
	require.NoError(t, err)
	for _, stage := range []string{"builder", "sender"} {
		t.Run(stage, func(t *testing.T) {
			calls := 0
			channel, err := mail.NewChannel(mail.SenderFunc(func(context.Context, mail.Envelope) error {
				calls++
				return want
			}))
			require.NoError(t, err)
			n := mailNotice{build: func(context.Context, notification.Recipient) (mail.Message, error) {
				if stage == "builder" {
					return mail.Message{}, want
				}
				return mail.Message{}, nil
			}}
			err = channel.Send(t.Context(), notification.Delivery{Recipient: to, Notification: n})
			require.ErrorIs(t, err, want)
			if stage == "builder" {
				assert.Zero(t, calls)
			} else {
				assert.Equal(t, 1, calls, "the channel does not retry")
			}
		})
	}
}

func TestMailChannelChecksCancellationBeforeSending(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	channel, err := mail.NewChannel(mail.SenderFunc(func(context.Context, mail.Envelope) error {
		calls++
		return nil
	}))
	require.NoError(t, err)
	to, err := mail.To("user@example.com")
	require.NoError(t, err)
	n := mailNotice{build: func(context.Context, notification.Recipient) (mail.Message, error) {
		cancel()
		return mail.Message{}, nil
	}}
	delivery := notification.Delivery{Recipient: to, Notification: n}
	require.ErrorIs(t, channel.Send(ctx, delivery), context.Canceled)
	require.ErrorIs(t, channel.Send(ctx, delivery), context.Canceled)
	_, err = to.MailRoute(ctx)
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, calls)
}

func TestMailConstruction(t *testing.T) {
	var sender mail.SenderFunc
	for _, value := range []mail.Sender{nil, sender} {
		channel, err := mail.NewChannel(value)
		require.ErrorIs(t, err, notification.ErrInvalidConfig)
		assert.Nil(t, channel)
	}
	for _, value := range []string{"", "not-an-address", "a@example.com, b@example.com", "a@example.com\r\nBcc: b@example.com"} {
		_, err := mail.To(value)
		require.ErrorIs(t, err, mail.ErrInvalidAddress)
	}
}
