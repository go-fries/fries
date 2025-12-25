package channel

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/queue/v3"
)

// Driver is an in-memory queue driver using Go channels and heap for priority
type Driver struct {
	mu       sync.RWMutex
	queues   map[string]*priorityQueue // queue name -> priority queue
	dead     map[string]*priorityQueue // dead letter queues
	notifyCh chan struct{}             // notification channel for new jobs

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

var (
	_ queue.Driver      = (*Driver)(nil)
	_ queue.Startable   = (*Driver)(nil)
	_ queue.Stoppable   = (*Driver)(nil)
	_ queue.Retryable   = (*Driver)(nil)
	_ queue.Inspectable = (*Driver)(nil)
)

// Option configures the Driver
type Option func(*Driver)

// New creates a new channel driver
func New(opts ...Option) *Driver {
	d := &Driver{
		queues:   make(map[string]*priorityQueue),
		dead:     make(map[string]*priorityQueue),
		notifyCh: make(chan struct{}, 1),
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// Start starts the driver's background tasks
func (d *Driver) Start(ctx context.Context) error {
	d.ctx, d.cancel = context.WithCancel(ctx)

	// Start the delayed job processor
	d.wg.Add(1)
	go d.processDelayed()

	return nil
}

// Stop stops the driver
func (d *Driver) Stop(ctx context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Push pushes a job to the queue
func (d *Driver) Push(ctx context.Context, job *queue.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	queueName := job.Queue()
	pq := d.getOrCreateQueue(queueName)

	heap.Push(pq, job)

	// Notify waiting consumers
	d.notify()

	return nil
}

// Pop retrieves a job from the queue (blocking)
func (d *Driver) Pop(ctx context.Context, queues ...string) (*queue.Job, error) {
	if len(queues) == 0 {
		queues = []string{"default"}
	}

	for {
		// Try to get a ready job
		if job := d.tryPop(queues); job != nil {
			return job, nil
		}

		// Wait for notification or context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-d.notifyCh:
			// New job available, try again
		case <-time.After(100 * time.Millisecond):
			// Periodic check for delayed jobs becoming ready
		}
	}
}

// tryPop attempts to pop a ready job from any of the specified queues
func (d *Driver) tryPop(queues []string) *queue.Job {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Try each queue in order, looking for the highest priority ready job
	var bestJob *queue.Job
	var bestQueue *priorityQueue

	for _, queueName := range queues {
		pq, exists := d.queues[queueName]
		if !exists || pq.Len() == 0 {
			continue
		}

		// Peek at the top job
		job := pq.Peek()
		if job == nil || job.AvailableAt().After(now) {
			continue
		}

		// Check if this job has higher priority than our current best
		if bestJob == nil || job.Priority() > bestJob.Priority() {
			bestJob = job
			bestQueue = pq
		}
	}

	if bestJob != nil {
		heap.Pop(bestQueue)
		return bestJob
	}

	return nil
}

// Ack acknowledges job completion
func (d *Driver) Ack(_ context.Context, _ *queue.Job) error {
	// For in-memory driver, job is already removed during Pop
	return nil
}

// Fail marks a job as failed
func (d *Driver) Fail(ctx context.Context, job *queue.Job, err error) error {
	job.IncrAttempts()
	job.SetFailed(err)

	if job.Attempts() >= job.MaxAttempts() {
		// Move to dead letter queue
		return d.moveToDead(job)
	}

	// Exponential backoff retry
	delay := time.Duration(job.Attempts()*job.Attempts()) * time.Second
	job.SetAvailableAt(time.Now().Add(delay))

	return d.Push(ctx, job)
}

// moveToDead moves a job to the dead letter queue
func (d *Driver) moveToDead(job *queue.Job) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	queueName := job.Queue()
	pq := d.getOrCreateDeadQueue(queueName)

	heap.Push(pq, job)

	return nil
}

// Size returns the number of pending jobs
func (d *Driver) Size(_ context.Context, queueName string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pq, exists := d.queues[queueName]
	if !exists {
		return 0, nil
	}

	return int64(pq.Len()), nil
}

// Dead returns the number of dead jobs
func (d *Driver) Dead(_ context.Context, queueName string) (int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pq, exists := d.dead[queueName]
	if !exists {
		return 0, nil
	}

	return int64(pq.Len()), nil
}

// Clear clears all jobs from the queue
func (d *Driver) Clear(_ context.Context, queueName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.queues, queueName)

	return nil
}

// ClearDead clears all jobs from the dead letter queue
func (d *Driver) ClearDead(_ context.Context, queueName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.dead, queueName)

	return nil
}

// Retry moves a job from dead letter queue back to main queue
func (d *Driver) Retry(ctx context.Context, queueName string, jobID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	pq, exists := d.dead[queueName]
	if !exists {
		return nil
	}

	// Find and remove the job from dead queue
	for i := 0; i < pq.Len(); i++ {
		if pq.items[i].ID() == jobID {
			job := heap.Remove(pq, i).(*queue.Job)

			// Reset attempts and re-queue
			mainQueue := d.getOrCreateQueue(queueName)
			job.SetAvailableAt(time.Now())
			heap.Push(mainQueue, job)

			d.notify()
			return nil
		}
	}

	return nil
}

// RetryAll moves all jobs from dead letter queue back to main queue
func (d *Driver) RetryAll(_ context.Context, queueName string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pq, exists := d.dead[queueName]
	if !exists {
		return 0, nil
	}

	count := int64(pq.Len())
	mainQueue := d.getOrCreateQueue(queueName)
	now := time.Now()

	for pq.Len() > 0 {
		job := heap.Pop(pq).(*queue.Job)
		job.SetAvailableAt(now)
		heap.Push(mainQueue, job)
	}

	if count > 0 {
		d.notify()
	}

	return count, nil
}

// Peek returns jobs from the queue without removing them
func (d *Driver) Peek(_ context.Context, queueName string, limit int) ([]*queue.Job, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pq, exists := d.queues[queueName]
	if !exists {
		return nil, nil
	}

	jobs := make([]*queue.Job, 0, min(limit, pq.Len()))
	for i := 0; i < limit && i < pq.Len(); i++ {
		jobs = append(jobs, pq.items[i])
	}

	return jobs, nil
}

// PeekDead returns jobs from the dead letter queue without removing them
func (d *Driver) PeekDead(_ context.Context, queueName string, limit int) ([]*queue.Job, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	pq, exists := d.dead[queueName]
	if !exists {
		return nil, nil
	}

	jobs := make([]*queue.Job, 0, min(limit, pq.Len()))
	for i := 0; i < limit && i < pq.Len(); i++ {
		jobs = append(jobs, pq.items[i])
	}

	return jobs, nil
}

// notify sends a notification that a new job is available
func (d *Driver) notify() {
	select {
	case d.notifyCh <- struct{}{}:
	default:
		// Channel already has notification pending
	}
}

// getOrCreateQueue gets or creates a priority queue
func (d *Driver) getOrCreateQueue(name string) *priorityQueue {
	pq, exists := d.queues[name]
	if !exists {
		pq = &priorityQueue{items: make([]*queue.Job, 0)}
		heap.Init(pq)
		d.queues[name] = pq
	}
	return pq
}

// getOrCreateDeadQueue gets or creates a dead letter queue
func (d *Driver) getOrCreateDeadQueue(name string) *priorityQueue {
	pq, exists := d.dead[name]
	if !exists {
		pq = &priorityQueue{items: make([]*queue.Job, 0)}
		heap.Init(pq)
		d.dead[name] = pq
	}
	return pq
}

// processDelayed periodically moves delayed jobs that are now ready
func (d *Driver) processDelayed() {
	defer d.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.checkDelayedJobs()
		}
	}
}

// checkDelayedJobs notifies if any delayed jobs are now ready
func (d *Driver) checkDelayedJobs() {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()

	for _, pq := range d.queues {
		if pq.Len() > 0 {
			job := pq.Peek()
			if job != nil && !job.AvailableAt().After(now) {
				d.notify()
				return
			}
		}
	}
}

// priorityQueue implements heap.Interface for priority-based job ordering
type priorityQueue struct {
	items []*queue.Job
}

func (pq *priorityQueue) Len() int { return len(pq.items) }

func (pq *priorityQueue) Less(i, j int) bool {
	// Higher priority first
	if pq.items[i].Priority() != pq.items[j].Priority() {
		return pq.items[i].Priority() > pq.items[j].Priority()
	}
	// Earlier availableAt first (FIFO for same priority)
	return pq.items[i].AvailableAt().Before(pq.items[j].AvailableAt())
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *priorityQueue) Push(x any) {
	pq.items = append(pq.items, x.(*queue.Job))
}

func (pq *priorityQueue) Pop() any {
	old := pq.items
	n := len(old)
	job := old[n-1]
	pq.items = old[0 : n-1]
	return job
}

// Peek returns the top item without removing it
func (pq *priorityQueue) Peek() *queue.Job {
	if len(pq.items) == 0 {
		return nil
	}
	return pq.items[0]
}
