package queue

import (
	"context"
	"sync"
)

// Worker consumes jobs from the queue
type Worker struct {
	driver      Driver
	handler     Handler
	queues      []string // queues to listen on
	concurrency int      // number of concurrent workers

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// WorkerOption configures a Worker
type WorkerOption func(*Worker)

// WithConcurrency sets the number of concurrent workers
func WithConcurrency(n int) WorkerOption {
	return func(w *Worker) {
		if n < 1 {
			n = 1
		}
		w.concurrency = n
	}
}

// WithQueues sets the queues to listen on
func WithQueues(queues ...string) WorkerOption {
	return func(w *Worker) {
		w.queues = queues
	}
}

// NewWorker creates a new worker
func NewWorker(driver Driver, handler Handler, opts ...WorkerOption) *Worker {
	w := &Worker{
		driver:      driver,
		handler:     handler,
		queues:      []string{"default"},
		concurrency: 1,
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) error {
	w.ctx, w.cancel = context.WithCancel(ctx)

	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.work(i)
	}

	return nil
}

// Stop gracefully stops the worker (waits for current jobs to complete)
func (w *Worker) Stop(ctx context.Context) error {
	if w.cancel != nil {
		w.cancel()
	}

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) work(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			job, err := w.driver.Pop(w.ctx, w.queues...)
			if err != nil {
				// Context cancelled or other error, check if we should exit
				select {
				case <-w.ctx.Done():
					return
				default:
					continue
				}
			}

			if job == nil {
				continue
			}

			if err := w.handler.Handle(w.ctx, job); err != nil {
				_ = w.driver.Fail(w.ctx, job, err)
			} else {
				_ = w.driver.Ack(w.ctx, job)
			}
		}
	}
}
