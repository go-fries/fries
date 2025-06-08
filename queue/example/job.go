package example

import (
	"context"
	"time"
)

type Job interface {
	ID() string

	Queue() string

	Subject() string

	Meta() map[string][]string

	Data() []byte

	Release(ctx context.Context) error

	Later(ctx context.Context, delay time.Duration) error

	LaterOn(ctx context.Context, delay time.Time) error
}

type DemoJob struct {
	Job
	ID string
}

func RunDemoJob(ctx context.Context, job *DemoJob) error {
	_ = job.Release(ctx)
	_ = job.Later(ctx, 5*time.Second)
	_ = job.LaterOn(ctx, time.Now().Add(10*time.Second))

	return nil
}

type QueueName string

type Queue interface {
	Push(ctx context.Context, job Job, opts ...Option) error

	Use(name QueueName) Queue
}

type options struct {
	id         string
	queue      string
	sync       bool
	maxRetries int
	meta       map[string][]string
}

type Option func(*options)

func WithQueue(queue string) Option {
	return func(opts *options) {
		opts.queue = queue
	}
}

func WithSync() Option {
	return func(opts *options) {
		opts.sync = true
	}
}

func WithMaxRetries(maxRetries int) Option {
	return func(opts *options) {
		opts.maxRetries = maxRetries
	}
}

func WithMeta(meta map[string][]string) Option {
	return func(opts *options) {
		opts.meta = meta
	}
}

func WithID(id string) Option {
	return func(opts *options) {
		opts.id = id
	}
}

func ExampleQueueUsage(ctx context.Context, queue Queue) error {
	job := &DemoJob{
		ID: "job-123",
	}

	if err := queue.Push(ctx, job,
		WithQueue("default"),
		WithSync(),
		WithMaxRetries(3),
		WithMeta(map[string][]string{
			"example-key": {"example-value"},
		}),
		WithID("xxxx"),
	); err != nil {
		return err
	}

	_ = queue.Use("xxx").Push(ctx, job)

	if err := RunDemoJob(ctx, job); err != nil {
		return err
	}

	return nil
}
