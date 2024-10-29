package crontab_test

import (
	"context"

	"github.com/flc1125/go-cron/v4"
	"github.com/go-kratos/kratos/v2"
	"github.com/redis/go-redis/v9"

	"github.com/go-kratos-ecosystem/components/v2/crontab"
	"github.com/go-kratos-ecosystem/components/v2/crontab/middleware/distributednooverlapping"
	"github.com/go-kratos-ecosystem/components/v2/crontab/middleware/distributednooverlapping/redismutex"
)

type testJob struct {
	distributednooverlapping.DefaultTTLJobWithMutex // optional, if you don't need to implement GetMutexTTL
}

var _ distributednooverlapping.JobWithMutex = (*testJob)(nil)

func (m *testJob) Run(context.Context) error {
	// do something
	return nil
}

func (m *testJob) GetMutexKey() string {
	return "test"
}

func Example() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	c := cron.New(
		cron.WithSeconds(),
		cron.WithMiddleware(
			distributednooverlapping.New(
				redismutex.New(rdb, redismutex.WithPrefix("cron:mutex")),
			),
		),
	)

	_, _ = c.AddJob("* * * * * *", &testJob{})

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
