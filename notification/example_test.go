package notification_test

import (
	"context"
	"fmt"
	netmail "net/mail"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
	"github.com/go-fries/fries/notification/v4/database/memory"
	"github.com/go-fries/fries/notification/v4/mail"
)

type invoiceRecipient struct {
	key     notification.RecipientKey
	address netmail.Address
}

func (r invoiceRecipient) NotificationKey() notification.RecipientKey { return r.key }

func (r invoiceRecipient) MailRoute(context.Context) (netmail.Address, error) {
	return r.address, nil
}

type invoicePaid struct {
	InvoiceID string
}

func (invoicePaid) Type() string { return "invoice.paid.v1" }

func (invoicePaid) Via(context.Context, notification.Recipient) ([]string, error) {
	return []string{mail.Name, database.Name}, nil
}

func (n invoicePaid) ToMail(context.Context, notification.Recipient) (mail.Message, error) {
	return mail.Message{
		Subject:  "Invoice paid",
		TextBody: fmt.Sprintf("Invoice %s has been paid.", n.InvoiceID),
	}, nil
}

func (n invoicePaid) ToDatabase(context.Context, notification.Recipient) (any, error) {
	return struct {
		InvoiceID string `json:"invoice_id"`
	}{InvoiceID: n.InvoiceID}, nil
}

func ExampleManager_Send() {
	if err := sendInvoice(context.Background()); err != nil {
		fmt.Println(err)
	}
	// Output:
	// mail accepted for taylor@example.com: Invoice paid
	// mail sent
	// database sent
	// unread: 1
	// unread after reading: 0
}

func sendInvoice(ctx context.Context) error {
	// The application supplies a mail transport. This example only prints.
	mailer := mail.SenderFunc(func(_ context.Context, envelope mail.Envelope) error {
		fmt.Printf("mail accepted for %s: %s\n", envelope.To.Address, envelope.Message.Subject)
		return nil
	})
	mailChannel, err := mail.NewChannel(mailer)
	if err != nil {
		return err
	}
	store := memory.New() // Process-local; no resources to close.
	databaseChannel, err := database.NewChannel(store)
	if err != nil {
		return err
	}
	notifier, err := notification.New(
		notification.WithChannel(mail.Name, mailChannel),
		notification.WithChannel(database.Name, databaseChannel),
	)
	if err != nil {
		return err
	}
	to := invoiceRecipient{
		key:     notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"},
		address: netmail.Address{Name: "Taylor", Address: "taylor@example.com"},
	}
	result, err := notifier.Send(ctx, to, invoicePaid{InvoiceID: "INV-1001"},
		notification.WithID("invoice-paid:INV-1001:user-42"),
	)
	if err != nil {
		// Inspect result.Deliveries before deciding what to retry: some channels
		// may already have succeeded. Retrying the entire Send can repeat mail.
		return err
	}
	for _, delivery := range result.Deliveries {
		fmt.Println(delivery.Channel, delivery.Status)
	}
	count, err := store.CountUnread(ctx, to.key)
	if err != nil {
		return err
	}
	fmt.Println("unread:", count)
	page, err := store.List(ctx, to.key, database.ListOptions{UnreadOnly: true})
	if err != nil {
		return err
	}
	for _, record := range page.Records {
		if err := store.MarkRead(ctx, to.key, record.ID); err != nil {
			return err
		}
	}
	count, err = store.CountUnread(ctx, to.key)
	if err != nil {
		return err
	}
	fmt.Println("unread after reading:", count)
	return nil
}
