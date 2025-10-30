package async

import (
	"context"
	"fmt"
	"sync"
)

type Message struct {
	Task string
	Args any
}

type Queue interface {
	Enqueue(ctx context.Context, message *Message) error
	Dequeue(ctx context.Context) (*Message, error)
}

type Handler func(ctx context.Context, args any) error

type Provider interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	Add(task string, handler Handler) error
	Submit(ctx context.Context, task string, args any) error
}

type handlerWrapper struct {
	task    string
	handler Handler

	workerNum int // number of workers for this handler
}

type provider struct {
	mu       sync.RWMutex
	handlers map[string]*handlerWrapper

	queue Queue

	stopCh   chan struct{}
	stopOnce sync.Once

	processWg sync.WaitGroup
}

func NewProvider(
	queue Queue,
) Provider {
	return &provider{
		queue:  queue,
		stopCh: make(chan struct{}),
	}
}

func (p *provider) Start(ctx context.Context) error {
	for {
		select {
		case <-p.stopCh:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
			// continue processing
		}

		message, err := p.queue.Dequeue(ctx)
		if err != nil {
			continue
		}

		p.mu.RLock()
		wrapper, exists := p.handlers[message.Task]
		p.mu.RUnlock()

		if !exists {
			// handler not found, skip
			continue
		}

		// execute handler
		p.processWg.Go(func() {
			if err := wrapper.handler(ctx, message.Args); err != nil {
				// handle error (e.g., log it)
			}
		})
	}
}

func (p *provider) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
	p.processWg.Wait()
	return nil
}

func (p *provider) Add(task string, handler Handler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.handlers == nil {
		p.handlers = make(map[string]*handlerWrapper)
	}

	if _, exists := p.handlers[task]; exists {
		return fmt.Errorf("task already exists")
	}

	p.handlers[task] = &handlerWrapper{
		task:    task,
		handler: handler,
	}
	return nil
}

func (p *provider) Submit(ctx context.Context, task string, args any) error {
	message := &Message{
		Task: task,
		Args: args,
	}
	return p.queue.Enqueue(ctx, message)
}
