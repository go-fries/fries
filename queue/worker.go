package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type workerConfig struct {
	queue             string
	concurrency       int
	pollInterval      time.Duration
	visibilityTimeout time.Duration
	handlerTimeout    time.Duration
	retryPolicy       RetryPolicy
	middleware        []Middleware
	handlers          map[string]Handler
}

type WorkerOption interface {
	apply(*workerConfig)
}

type workerOptionFunc func(*workerConfig)

func (f workerOptionFunc) apply(c *workerConfig) {
	f(c)
}

func WithWorkerQueue(name string) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if name != "" {
			c.queue = name
		}
	})
}

func WithConcurrency(concurrency int) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if concurrency > 0 {
			c.concurrency = concurrency
		}
	})
}

func WithPollInterval(interval time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if interval > 0 {
			c.pollInterval = interval
		}
	})
}

func WithVisibilityTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.visibilityTimeout = timeout
		}
	})
}

func WithHandlerTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.handlerTimeout = timeout
		}
	})
}

func WithRetryPolicy(policy RetryPolicy) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if policy != nil {
			c.retryPolicy = policy
		}
	})
}

func WithMiddleware(middleware ...Middleware) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		c.middleware = append(c.middleware, middleware...)
	})
}

func Handle(taskType string, handler Handler) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if taskType != "" && handler != nil {
			c.handlers[taskType] = handler
		}
	})
}

func newWorkerConfig(opts ...WorkerOption) *workerConfig {
	c := &workerConfig{
		queue:             DefaultQueue,
		concurrency:       1,
		pollInterval:      time.Second,
		visibilityTimeout: time.Minute,
		retryPolicy:       FixedRetry(3, time.Second),
		handlers:          make(map[string]Handler),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

type Worker struct {
	backend Backend
	config  *workerConfig
}

func NewWorker(backend Backend, opts ...WorkerOption) *Worker {
	return &Worker{
		backend: backend,
		config:  newWorkerConfig(opts...),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, w.config.concurrency)
	var wg sync.WaitGroup

	for i := 0; i < w.config.concurrency; i++ {
		wg.Go(func() {
			if err := w.loop(ctx); err != nil {
				errs <- err
				cancel()
			}
		})
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		select {
		case err := <-errs:
			return err
		default:
			return nil
		}
	case err := <-errs:
		cancel()
		<-done
		return err
	}
}

func (w *Worker) loop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		lease, err := w.backend.Dequeue(ctx, w.config.queue, w.config.visibilityTimeout)
		if errors.Is(err, ErrNoTask) {
			if err := sleep(ctx, w.config.pollInterval); err != nil {
				return nil
			}
			continue
		}
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
		if lease == nil || lease.Task == nil {
			continue
		}

		if err := w.process(ctx, lease); err != nil {
			return err
		}
	}
}

func (w *Worker) process(ctx context.Context, lease *Lease) error {
	handler, ok := w.config.handlers[lease.Task.Type]
	if !ok {
		return w.backend.DeadLetter(ctx, lease, ErrHandlerNotFound.Error())
	}

	err := w.handle(ctx, handler, lease.Task)
	if err == nil {
		return w.backend.Ack(ctx, lease)
	}

	delay, ok := w.config.retryPolicy.NextDelay(lease.Task, err)
	if !ok {
		return w.backend.DeadLetter(ctx, lease, fmt.Sprintf("%s: %v", ErrRetryExhausted, err))
	}

	return w.backend.Retry(ctx, lease, delay)
}

func (w *Worker) handle(ctx context.Context, handler Handler, task *Task) error {
	handler = chain(handler, w.config.middleware)
	if w.config.handlerTimeout <= 0 {
		return handler.Handle(ctx, task)
	}

	handlerCtx, cancel := context.WithTimeout(ctx, w.config.handlerTimeout)
	defer cancel()
	return handler.Handle(handlerCtx, task)
}

func sleep(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
