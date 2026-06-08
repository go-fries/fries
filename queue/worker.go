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

	mu        sync.Mutex
	stop      context.CancelFunc
	interrupt context.CancelFunc
	done      chan struct{}
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
	pollCtx, stop := context.WithCancel(ctx)
	handlerCtx, interrupt := context.WithCancel(context.WithoutCancel(ctx))
	lifecycleDone := make(chan struct{})
	if !w.start(stop, interrupt, lifecycleDone) {
		stop()
		interrupt()
		return errors.New("queue: worker already running")
	}
	defer func() {
		stop()
		interrupt()
		w.finish(lifecycleDone)
	}()

	stopWatchingRun := context.AfterFunc(ctx, func() {
		stop()
		interrupt()
	})
	defer stopWatchingRun()

	errs := make(chan error, w.config.concurrency)
	var wg sync.WaitGroup

	for i := 0; i < w.config.concurrency; i++ {
		wg.Go(func() {
			if err := w.loop(pollCtx, handlerCtx); err != nil {
				errs <- err
				stop()
				interrupt()
			}
		})
	}

	workersDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		select {
		case err := <-errs:
			return err
		default:
			return nil
		}
	case err := <-errs:
		stop()
		interrupt()
		<-workersDone
		return err
	}
}

// Stop stops polling for new tasks and waits for in-flight tasks to finish.
//
// If ctx is canceled before the worker exits, Stop cancels in-flight task
// handlers and returns ctx.Err().
func (w *Worker) Stop(ctx context.Context) error {
	stop, interrupt, done := w.lifecycle()
	if stop == nil {
		return nil
	}

	stop()
	select {
	case <-done:
		return nil
	default:
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		interrupt()
		return ctx.Err()
	}
}

func (w *Worker) lifecycle() (context.CancelFunc, context.CancelFunc, chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.stop, w.interrupt, w.done
}

func (w *Worker) start(stop, interrupt context.CancelFunc, done chan struct{}) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.done != nil {
		return false
	}
	w.stop = stop
	w.interrupt = interrupt
	w.done = done
	return true
}

func (w *Worker) finish(done chan struct{}) {
	w.mu.Lock()
	if w.done == done {
		w.stop = nil
		w.interrupt = nil
		w.done = nil
	}
	close(done)
	w.mu.Unlock()
}

func (w *Worker) loop(pollCtx, handlerCtx context.Context) error {
	for {
		select {
		case <-pollCtx.Done():
			return nil
		default:
		}

		lease, err := w.queue.Dequeue(pollCtx, w.config.queue, w.config.visibilityTimeout)
		if errors.Is(err, ErrNoTask) {
			if err := sleep(pollCtx, w.config.pollInterval); err != nil {
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

		if err := w.process(handlerCtx, lease); err != nil {
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
