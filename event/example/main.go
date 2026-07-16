package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-fries/fries/event/middleware/recovery/v4"
	"github.com/go-fries/fries/event/v4"
)

type userCreated struct {
	Name string
}

type welcomeHandler struct{}

func (welcomeHandler) Handle(_ context.Context, value userCreated) error {
	fmt.Println("welcome", value.Name)
	return nil
}

func logging(next event.Next) event.Next {
	return func(ctx context.Context, value any) error {
		err := next(ctx, value)
		if err != nil {
			slog.ErrorContext(ctx, "event handler failed", "event", value, "error", err)
		}
		return err
	}
}

func main() {
	dispatcher := event.New(
		event.WithMiddleware(logging, recovery.New()),
	)

	subscription := dispatcher.Subscribe(
		event.HandlerFor[userCreated](welcomeHandler{}),
		event.HandlerFor[userCreated](event.HandlerFunc[userCreated](
			func(_ context.Context, value userCreated) error {
				fmt.Println("created user", value.Name)
				return nil
			},
		)),
	)
	defer subscription.Unsubscribe()

	if err := dispatcher.Dispatch(
		context.Background(),
		userCreated{Name: "ZhangSan"},
		event.WithConcurrency(2),
		event.ContinueOnError(),
	); err != nil {
		slog.Error("dispatch event", "error", err)
	}
}
