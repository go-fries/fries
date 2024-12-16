package crontab_test

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/go-kratos-ecosystem/components/v2/crontab"
	"github.com/go-kratos/kratos/v2"
)

func Example() {
	c := cron.New(
		cron.WithSeconds(),
		cron.WithMiddleware( /*...*/ ),
	)

	_, _ = c.AddFunc("* * * * * *", func(context.Context) error {
		// do something
		return nil
	})

	// kratos app start
	app := kratos.New(
		kratos.Server(
			crontab.NewServer(c),
		),
	)

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
