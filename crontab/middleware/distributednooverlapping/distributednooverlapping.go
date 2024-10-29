package distributednooverlapping

import (
	"context"

	"github.com/flc1125/go-cron/v4"
)

type options struct {
	mu     Mutex
	logger cron.Logger
}

type Option func(*options)

func WithLogger(logger cron.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func newOptions(mu Mutex, opts ...Option) options {
	opt := options{
		mu:     mu,
		logger: cron.DefaultLogger,
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func New(mu Mutex, opts ...Option) cron.Middleware {
	o := newOptions(mu, opts...)
	return func(original cron.Job) cron.Job {
		return cron.JobFunc(func(ctx context.Context) error {
			job, ok := interface{}(original).(JobWithMutex)
			if !ok {
				return original.Run(ctx)
			}

			if success, err := o.mu.Lock(ctx, job); err != nil {
				o.logger.Error(err, "failed to lock mutex", "key", job.GetMutexKey())
				return err
			} else if success {
				defer func() {
					_ = o.mu.Unlock(ctx, job)
				}()
				return original.Run(ctx)
			} else {
				o.logger.Info("skip job [%s], because distributed no overlapping", "key", job.GetMutexKey())
				return nil
			}
		})
	}
}
