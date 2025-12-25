package queue

import (
	"context"
	"sync"
	"time"
)

// Manager is the queue manager that coordinates drivers and workers
type Manager struct {
	driver  Driver
	workers map[string]*Worker // queue name -> worker
	mu      sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
}

// ManagerOption configures a Manager
type ManagerOption func(*Manager)

// NewManager creates a new queue manager
func NewManager(driver Driver, opts ...ManagerOption) *Manager {
	m := &Manager{
		driver:  driver,
		workers: make(map[string]*Worker),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Push pushes a job to the queue
func (m *Manager) Push(ctx context.Context, job Job) error {
	return m.driver.Push(ctx, job)
}

// PushOn is a convenience method to push a job to a specific queue
func (m *Manager) PushOn(ctx context.Context, queue string, payload any, opts ...JobOption) error {
	opts = append([]JobOption{WithQueue(queue)}, opts...)
	job := NewJob(payload, opts...)
	return m.driver.Push(ctx, job)
}

// Later pushes a job with a delay
func (m *Manager) Later(ctx context.Context, delay time.Duration, job Job) error {
	job.SetAvailableAt(time.Now().Add(delay))
	return m.driver.Push(ctx, job)
}

// Size returns the number of pending jobs in the queue
func (m *Manager) Size(ctx context.Context, queue string) (int64, error) {
	return m.driver.Size(ctx, queue)
}

// Dead returns the number of jobs in the dead letter queue
func (m *Manager) Dead(ctx context.Context, queue string) (int64, error) {
	return m.driver.Dead(ctx, queue)
}

// Register registers a handler for a queue
func (m *Manager) Register(queue string, handler Handler, opts ...WorkerOption) {
	m.mu.Lock()
	defer m.mu.Unlock()

	opts = append([]WorkerOption{WithQueues(queue)}, opts...)
	worker := NewWorker(m.driver, handler, opts...)
	m.workers[queue] = worker
}

// Start starts the manager and all registered workers
// Implements contract.Server interface
func (m *Manager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Start the driver if it implements Startable
	if startable, ok := m.driver.(Startable); ok {
		if err := startable.Start(m.ctx); err != nil {
			return err
		}
	}

	// Start all workers
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, worker := range m.workers {
		if err := worker.Start(m.ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop gracefully stops the manager and all workers
// Implements contract.Server interface
func (m *Manager) Stop(ctx context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Stop all workers
	for _, worker := range m.workers {
		if err := worker.Stop(ctx); err != nil {
			return err
		}
	}

	// Stop the driver if it implements Stoppable
	if stoppable, ok := m.driver.(Stoppable); ok {
		if err := stoppable.Stop(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Driver returns the underlying driver
func (m *Manager) Driver() Driver {
	return m.driver
}
