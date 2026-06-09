package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/queue/v3"
)

// Queue is an in-memory queue implementation for tests and local use.
type Queue struct {
	mu         sync.Mutex
	queues     map[string][]*queue.Task
	deadLetter map[string][]*queue.Task
	notify     chan struct{}
}

var _ queue.Queue = (*Queue)(nil)

// NewQueue creates an empty in-memory queue.
func NewQueue() *Queue {
	return &Queue{
		queues:     make(map[string][]*queue.Task),
		deadLetter: make(map[string][]*queue.Task),
		notify:     make(chan struct{}),
	}
}

// Enqueue stores task in memory.
func (q *Queue) Enqueue(ctx context.Context, task *queue.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task = task.Clone()
	if task.Queue == "" {
		task.Queue = queue.DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.queues[task.Queue] = append(q.queues[task.Queue], task)
	q.signal()
	return nil
}

// NewConsumer creates a consumer using config.
func (q *Queue) NewConsumer(ctx context.Context, config queue.ConsumerConfig) (queue.Consumer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config = config.Normalize()
	return &consumer{
		queue: q,
		name:  config.Queue,
		done:  make(chan struct{}),
	}, nil
}

type consumer struct {
	queue *Queue
	name  string
	done  chan struct{}
	once  sync.Once
}

func (c *consumer) Receive(ctx context.Context) (queue.Delivery, error) {
	for {
		task, notify, wait := c.queue.next(c.name)
		if task != nil {
			return &delivery{queue: c.queue, task: task}, nil
		}

		var timer *time.Timer
		var timerC <-chan time.Time
		if wait > 0 {
			timer = time.NewTimer(wait)
			timerC = timer.C
		}

		select {
		case <-ctx.Done():
			stopTimer(timer)
			return nil, ctx.Err()
		case <-c.done:
			stopTimer(timer)
			return nil, queue.ErrConsumerClosed
		case <-notify:
			stopTimer(timer)
		case <-timerC:
		}
	}
}

func (c *consumer) Close() error {
	c.once.Do(func() {
		close(c.done)
	})
	return nil
}

type delivery struct {
	queue *Queue
	task  *queue.Task
}

func (d *delivery) Task() *queue.Task {
	if d == nil {
		return nil
	}
	return d.task
}

func (*delivery) Ack(ctx context.Context) error {
	return ctx.Err()
}

func (d *delivery) Retry(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.Clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	return d.queue.Enqueue(ctx, task)
}

func (d *delivery) DeadLetter(ctx context.Context, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.Clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["queue.dead_letter.reason"] = reason

	d.queue.mu.Lock()
	defer d.queue.mu.Unlock()

	d.queue.deadLetter[task.Queue] = append(d.queue.deadLetter[task.Queue], task)
	return nil
}

func (q *Queue) next(queueName string) (*queue.Task, <-chan struct{}, time.Duration) {
	now := time.Now().UTC()
	q.mu.Lock()
	defer q.mu.Unlock()

	var wait time.Duration
	tasks := q.queues[queueName]
	for i, task := range tasks {
		if task.AvailableAt.After(now) {
			delay := task.AvailableAt.Sub(now)
			if wait == 0 || delay < wait {
				wait = delay
			}
			continue
		}

		q.queues[queueName] = append(tasks[:i], tasks[i+1:]...)
		task = task.Clone()
		task.Attempt++
		return task, q.notify, 0
	}

	return nil, q.notify, wait
}

func (q *Queue) signal() {
	close(q.notify)
	q.notify = make(chan struct{})
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// DeadLetters returns a copy of dead-lettered tasks for queueName.
func (q *Queue) DeadLetters(queueName string) []*queue.Task {
	if queueName == "" {
		queueName = queue.DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tasks := q.deadLetter[queueName]
	cloned := make([]*queue.Task, 0, len(tasks))
	for _, task := range tasks {
		cloned = append(cloned, task.Clone())
	}
	return cloned
}
