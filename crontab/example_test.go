package crontab_test

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/go-fries/fries/crontab/v3"
)

func ExampleNewServer() {
	server := crontab.NewServer(cron.New(cron.WithSeconds()))
	_, _ = server.Cron().AddFunc("* * * * * *", func(context.Context) error {
		return nil
	})

	_ = server
}
