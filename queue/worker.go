package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-fries/fries/codec/v3"
)

const (
	defaultConcurrency       = 1
	defaultPollInterval      = time.Second
	defaultVisibilityTimeout = 5 * time.Minute
	defaultRetryMaxAttempts  = 3
	defaultRetryDelay        = time.Second
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

// WorkerOption configures a Worker.
type WorkerOption interface {
	apply(*workerConfig)
}

type workerOptionFunc func(*workerConfig)

func (f workerOptionFunc) apply(c *workerConfig) {
	f(c)
}

// WithWorkerQueue sets the queue name consumed by the worker.
func WithWorkerQueue(name string) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if name != "" {
			c.queue = name
		}
	})
}

// WithConcurrency sets the number of concurrent worker loops.
func WithConcurrency(concurrency int) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if concurrency > 0 {
			c.concurrency = concurrency
		}
	})
}

// WithPollInterval sets how long the worker waits after an empty dequeue.
func WithPollInterval(interval time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if interval > 0 {
			c.pollInterval = interval
		}
	})
}

// WithVisibilityTimeout sets how long a leased task can remain unacknowledged before redelivery.
func WithVisibilityTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.visibilityTimeout = timeout
		}
	})
}

// WithHandlerTimeout sets a per-task handler timeout.
func WithHandlerTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.handlerTimeout = timeout
		}
	})
}

// WithRetryPolicy sets the retry policy used after handler errors.
func WithRetryPolicy(policy RetryPolicy) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if policy != nil {
			c.retryPolicy = policy
		}
	})
}

// WithMiddleware appends worker middleware around task handlers.
func WithMiddleware(middleware ...Middleware) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		c.middleware = append(c.middleware, middleware...)
	})
}

// Handle registers handler for taskType.
func Handle(taskType string, handler Handler) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if taskType != "" && handler != nil {
			c.handlers[taskType] = handler
		}
	})
}

// HandleFor decodes task payloads with the default JSON codec before calling handler.
func HandleFor[T any](taskType string, handler HandlerFor[T]) WorkerOption {
	return HandleForWithCodec(taskType, defaultCodec, handler)
}

// HandleForWithCodec decodes task payloads with codec before calling handler.
func HandleForWithCodec[T any](taskType string, codec codec.Codec, handler HandlerFor[T]) WorkerOption {
	if handler == nil {
		return Handle(taskType, nil)
	}
	if codec == nil {
		codec = defaultCodec
	}

	return Handle(taskType, HandlerFunc(func(ctx context.Context, task *Task) error {
		var payload T
		if err := codec.Unmarshal(task.Payload, &payload); err != nil {
			return err
		}
		return handler.Handle(ctx, &TaskFor[T]{
			Task:    task,
			Payload: payload,
		})
	}))
}

// HandleTasker registers tasker as the typed handler for its TaskType.
//
// It is equivalent to calling HandleFor(tasker.TaskType(), tasker). A nil tasker
// is ignored, matching HandleFor's nil-handler behavior.
func HandleTasker[T any](tasker Tasker[T]) WorkerOption {
	if tasker == nil {
		return Handle("", nil)
	}
	return HandleFor(tasker.TaskType(), tasker)
}

func newWorkerConfig(opts ...WorkerOption) *workerConfig {
	c := &workerConfig{
		queue:             DefaultQueue,
		concurrency:       defaultConcurrency,
		pollInterval:      defaultPollInterval,
		visibilityTimeout: defaultVisibilityTimeout,
		retryPolicy:       FixedRetry(defaultRetryMaxAttempts, defaultRetryDelay),
		handlers:          make(map[string]Handler),
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Worker consumes tasks from a queue and dispatches them to registered handlers.
type Worker struct {
	queue  Queue
	config *workerConfig
}

// NewWorker creates a worker with the provided queue and options.
func NewWorker(q Queue, opts ...WorkerOption) *Worker {
	return &Worker{
		queue:  q,
		config: newWorkerConfig(opts...),
	}
}

// Run starts worker loops and blocks until ctx is canceled or a queue operation fails.
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

		lease, err := w.queue.Dequeue(ctx, w.config.queue, w.config.visibilityTimeout)
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
		if lease == nil {
			continue
		}
		task := lease.Task()
		if task == nil {
			continue
		}

		if err := w.process(ctx, lease); err != nil {
			return err
		}
	}
}

func (w *Worker) process(ctx context.Context, lease Lease) error {
	task := lease.Task()
	handler, ok := w.config.handlers[task.Type]
	if !ok {
		return w.queue.DeadLetter(ctx, lease, ErrHandlerNotFound.Error())
	}

	err := w.handle(ctx, handler, task)
	if err == nil {
		return w.queue.Ack(ctx, lease)
	}

	delay, ok := w.config.retryPolicy.NextDelay(task, err)
	if !ok {
		return w.queue.DeadLetter(ctx, lease, fmt.Sprintf("%s: %v", ErrRetryExhausted, err))
	}

	return w.queue.Retry(ctx, lease, delay)
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
