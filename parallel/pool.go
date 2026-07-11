package parallel

import (
	"context"
	"sync"
)

type poolConfig struct {
	queueSize int
}

// PoolOption configures a Pool.
type PoolOption interface {
	apply(*poolConfig)
}

type poolOptionFunc func(*poolConfig)

func (f poolOptionFunc) apply(config *poolConfig) {
	f(config)
}

// WithQueueSize configures the number of tasks that may wait for a worker.
//
// A size of zero creates an unbuffered queue. Negative values keep the default
// queue size, which equals the worker count.
func WithQueueSize(size int) PoolOption {
	return poolOptionFunc(func(config *poolConfig) {
		if size >= 0 {
			config.queueSize = size
		}
	})
}

func newPoolConfig(workers int, options ...PoolOption) poolConfig {
	config := poolConfig{queueSize: workers}
	for _, option := range options {
		option.apply(&config)
	}

	return config
}

// Future represents the eventual result of a task accepted by a Pool.
//
// A Future may be waited on multiple times. Canceling a Wait context stops
// only that wait; it does not cancel the task. Future values are created by a
// Pool; the zero value is not valid.
type Future struct {
	done chan struct{}
	err  error
}

func newFuture() *Future {
	return &Future{done: make(chan struct{})}
}

// Done returns a channel that is closed when the task finishes.
func (f *Future) Done() <-chan struct{} {
	return f.done
}

// Wait waits for the task to finish or ctx to be canceled.
func (f *Future) Wait(ctx context.Context) error {
	select {
	case <-f.done:
		return f.err
	default:
	}

	select {
	case <-f.done:
		return f.err
	case <-ctx.Done():
		select {
		case <-f.done:
			return f.err
		default:
			return context.Cause(ctx)
		}
	}
}

func (f *Future) complete(err error) {
	f.err = err
	close(f.done)
}

type poolTask struct {
	ctx    context.Context
	task   Task
	future *Future
}

// Pool executes intermittently submitted tasks with a fixed number of
// long-lived workers and a bounded queue.
//
// A Pool must be shut down when it is no longer needed. Task lifetimes are
// controlled by the contexts passed to Submit and Execute.
type Pool struct {
	tasks   chan poolTask
	closing chan struct{}
	done    chan struct{}

	mu         sync.Mutex
	closed     bool
	stopOnce   sync.Once
	submitters sync.WaitGroup
	workers    sync.WaitGroup
}

// NewPool starts a fixed number of worker goroutines.
//
// NewPool panics if workers is not positive. By default, the queue can hold one
// task per worker; use WithQueueSize to change that capacity.
func NewPool(workers int, options ...PoolOption) *Pool {
	if workers <= 0 {
		panic("parallel: worker count must be greater than zero")
	}

	config := newPoolConfig(workers, options...)

	pool := &Pool{
		tasks:   make(chan poolTask, config.queueSize),
		closing: make(chan struct{}),
		done:    make(chan struct{}),
	}
	pool.workers.Add(workers)
	for range workers {
		go pool.work()
	}

	return pool
}

// Submit adds task to the pool and returns once it has been accepted.
//
// Submit applies backpressure when the queue is full. In that case it waits
// until a worker accepts the task, ctx is canceled, or the pool is shut down.
// The same ctx is passed to task, so asynchronous callers must provide a
// context whose lifetime is long enough for the task.
func (p *Pool) Submit(ctx context.Context, task Task) (*Future, error) {
	if task == nil {
		return nil, ErrNilTask
	}
	if err := context.Cause(ctx); err != nil {
		return nil, err
	}

	future := newFuture()
	item := poolTask{ctx: ctx, task: task, future: future}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()

		return nil, ErrPoolClosed
	}
	p.submitters.Add(1)
	p.mu.Unlock()
	defer p.submitters.Done()

	select {
	case p.tasks <- item:
		return future, nil
	case <-p.closing:
		return nil, ErrPoolClosed
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	}
}

// Execute submits task and waits for it to finish.
func (p *Pool) Execute(ctx context.Context, task Task) error {
	future, err := p.Submit(ctx, task)
	if err != nil {
		return err
	}

	return future.Wait(ctx)
}

// Shutdown stops accepting new tasks and waits for accepted tasks to finish.
//
// Canceling ctx stops only the wait; shutdown continues in the background.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.stop()

	select {
	case <-p.done:
		return nil
	default:
	}

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		select {
		case <-p.done:
			return nil
		default:
			return context.Cause(ctx)
		}
	}
}

func (p *Pool) stop() {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.closed = true
		close(p.closing)
		p.mu.Unlock()

		go func() {
			p.submitters.Wait()
			close(p.tasks)
			p.workers.Wait()
			close(p.done)
		}()
	})
}

func (p *Pool) work() {
	defer p.workers.Done()

	for item := range p.tasks {
		p.execute(item)
	}
}

func (p *Pool) execute(item poolTask) {
	if err := context.Cause(item.ctx); err != nil {
		item.future.complete(err)

		return
	}

	err := item.task(item.ctx)
	if err == nil {
		err = context.Cause(item.ctx)
	}
	item.future.complete(err)
}
