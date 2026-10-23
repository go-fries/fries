package notification_test

import (
	"context"
	netmail "net/mail"
	"testing"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
	"github.com/go-fries/fries/notification/v4/database/memory"
	"github.com/go-fries/fries/notification/v4/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelDeduplicationDoesNotImplyExactlyOnceMail(t *testing.T) {
	mailCalls := 0
	mailChannel, err := mail.NewChannel(mail.SenderFunc(func(context.Context, mail.Envelope) error {
		mailCalls++
		return nil
	}))
	require.NoError(t, err)
	store := memory.New()
	databaseChannel, err := database.NewChannel(store)
	require.NoError(t, err)
	notifier, err := notification.New(
		notification.WithChannel(mail.Name, mailChannel),
		notification.WithChannel(database.Name, databaseChannel),
	)
	require.NoError(t, err)
	to := invoiceRecipient{
		key:     notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"},
		address: netmail.Address{Address: "user@example.com"},
	}
	n := invoicePaid{InvoiceID: "INV-42"}
	first, err := notifier.Send(t.Context(), to, n, notification.WithID("stable"))
	require.NoError(t, err)
	require.NoError(t, store.MarkRead(t.Context(), to.key, first.Deliveries[1].ID))
	second, err := notifier.Send(t.Context(), to, n, notification.WithID("stable"))
	require.NoError(t, err)
	assert.Equal(t, first.Deliveries[0].ID, second.Deliveries[0].ID)
	assert.Equal(t, first.Deliveries[1].ID, second.Deliveries[1].ID)
	assert.Equal(t, 2, mailCalls)
	page, err := store.List(t.Context(), to.key, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.NotNil(t, page.Records[0].ReadAt)

	// Reusing an ID for different business data is unsupported. The inbox
	// detects the conflict, but the earlier mail cannot be rolled back.
	conflict, err := notifier.Send(t.Context(), to, invoicePaid{InvoiceID: "INV-different"}, notification.WithID("stable"))
	require.ErrorIs(t, err, database.ErrConflict)
	assert.Equal(t, notification.StatusSent, conflict.Deliveries[0].Status)
	assert.Equal(t, notification.StatusFailed, conflict.Deliveries[1].Status)
	assert.Equal(t, 3, mailCalls)
}
