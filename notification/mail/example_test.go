package mail_test

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/mail"
)

func ExampleTo() {
	to, err := mail.To("Taylor <taylor@example.com>")
	if err != nil {
		fmt.Println(err)
		return
	}
	channel, err := mail.NewChannel(mail.SenderFunc(func(_ context.Context, envelope mail.Envelope) error {
		fmt.Println(envelope.To.Address, envelope.Message.Subject)
		return nil
	}))
	if err != nil {
		fmt.Println(err)
		return
	}
	notifier, err := notification.New(notification.WithChannel(mail.Name, channel))
	if err != nil {
		fmt.Println(err)
		return
	}
	if _, err := notifier.Send(context.Background(), to, welcome{}); err != nil {
		fmt.Println(err)
	}
	// Output: taylor@example.com Welcome
}

type welcome struct{}

func (welcome) Type() string { return "welcome.v1" }

func (welcome) Via(context.Context, notification.Recipient) ([]string, error) {
	return []string{mail.Name}, nil
}

func (welcome) ToMail(context.Context, notification.Recipient) (mail.Message, error) {
	return mail.Message{Subject: "Welcome", TextBody: "Thanks for joining."}, nil
}
