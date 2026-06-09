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
	defaultRetryMaxAttempts  = 3
	defaultRetryDelay        = time.Second
	defaultSettlementTimeout = 30 * time.Second
)

type workerConfig struct {
	queue             string
	consumerName      string
	concurrency       int
	handlerTimeout    time.Duration
	settlementTimeout time.Duration
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

// WithConsumerName sets the backend consumer identity used by the worker.
func WithConsumerName(name string) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if name != "" {
			c.consumerName = name
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

// WithHandlerTimeout sets a per-task handler timeout.
func WithHandlerTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.handlerTimeout = timeout
		}
	})
}

// WithSettlementTimeout sets the timeout for Ack, Retry, and DeadLetter operations.
func WithSettlementTimeout(timeout time.Duration) WorkerOption {
	return workerOptionFunc(func(c *workerConfig) {
		if timeout > 0 {
			c.settlementTimeout = timeout
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
		settlementTimeout: defaultSettlementTimeout,
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
//
// Queue operation errors stop the worker and are returned to the caller unless
// they are normal stop signals such as context cancellation or ErrConsumerClosed.
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
			if err := w.loop(ctx, pollCtx, handlerCtx); err != nil {
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

func (w *Worker) loop(runCtx, pollCtx, handlerCtx context.Context) error {
	consumer, err := w.queue.NewConsumer(pollCtx, ConsumerConfig{
		Queue: w.config.queue,
		Name:  w.config.consumerName,
	})
	if err != nil {
		if isStopError(err) {
			return nil
		}
		return err
	}
	defer func() {
		_ = consumer.Close()
	}()

	for {
		select {
		case <-pollCtx.Done():
			return nil
		default:
		}

		delivery, err := consumer.Receive(pollCtx)
		if err != nil {
			if isStopError(err) {
				return nil
			}
			return err
		}
		if delivery == nil {
			continue
		}
		task := delivery.Task()
		if task == nil {
			continue
		}

		if err := w.process(handlerCtx, delivery); err != nil {
			if runCtx.Err() != nil && isStopError(err) {
				return nil
			}
			return err
		}
	}
}

func isStopError(err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrConsumerClosed)
}

func (w *Worker) process(ctx context.Context, delivery Delivery) error {
	task := delivery.Task()
	handler, ok := w.config.handlers[task.Type]
	if !ok {
		return w.deadLetter(ctx, delivery, ErrHandlerNotFound.Error())
	}

	err := w.handle(ctx, handler, task)
	if err == nil {
		return w.ack(ctx, delivery)
	}
	if errors.Is(err, ErrDiscard) {
		return w.ack(ctx, delivery)
	}
	if reason, ok := deadLetterReason(err); ok {
		return w.deadLetter(ctx, delivery, reason)
	}
	if retryAfter, ok := retryAfterDelay(err); ok {
		if _, ok := w.config.retryPolicy.NextDelay(task, err); !ok {
			return w.deadLetter(ctx, delivery, fmt.Sprintf("%s: %v", ErrRetryExhausted, err))
		}
		return w.retry(ctx, delivery, retryAfter)
	}

	delay, ok := w.config.retryPolicy.NextDelay(task, err)
	if !ok {
		return w.deadLetter(ctx, delivery, fmt.Sprintf("%s: %v", ErrRetryExhausted, err))
	}

	return w.retry(ctx, delivery, delay)
}

func (w *Worker) ack(ctx context.Context, delivery Delivery) error {
	return w.withSettlementContext(ctx, func(ctx context.Context) error {
		return delivery.Ack(ctx)
	})
}

func (w *Worker) retry(ctx context.Context, delivery Delivery, delay time.Duration) error {
	return w.withSettlementContext(ctx, func(ctx context.Context) error {
		return delivery.Retry(ctx, delay)
	})
}

func (w *Worker) deadLetter(ctx context.Context, delivery Delivery, reason string) error {
	return w.withSettlementContext(ctx, func(ctx context.Context) error {
		return delivery.DeadLetter(ctx, reason)
	})
}

func (w *Worker) withSettlementContext(ctx context.Context, fn func(context.Context) error) error {
	if w.config.settlementTimeout <= 0 {
		return fn(ctx)
	}

	settlementCtx, cancel := context.WithTimeout(ctx, w.config.settlementTimeout)
	defer cancel()
	return fn(settlementCtx)
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
