package queue

import (
	"context"
	"sync"
	"time"
)

type testQueue struct {
	mu         sync.Mutex
	queues     map[string][]*Task
	deadLetter map[string][]*Task
	notify     chan struct{}
}

func newTestQueue() *testQueue {
	return &testQueue{
		queues:     make(map[string][]*Task),
		deadLetter: make(map[string][]*Task),
		notify:     make(chan struct{}),
	}
}

func (q *testQueue) Enqueue(ctx context.Context, task *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	task = task.clone()
	if task.Queue == "" {
		task.Queue = DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.queues[task.Queue] = append(q.queues[task.Queue], task)
	q.signal()
	return nil
}

func (q *testQueue) NewConsumer(ctx context.Context, config ConsumerConfig) (Consumer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config = config.Normalize()
	return &testConsumer{
		queue: q,
		name:  config.Queue,
		done:  make(chan struct{}),
	}, nil
}

func (q *testQueue) Receive(ctx context.Context, queueName string) (Delivery, error) {
	consumer, err := q.NewConsumer(ctx, ConsumerConfig{Queue: queueName})
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = consumer.Close()
	}()
	return consumer.Receive(ctx)
}

type testConsumer struct {
	queue *testQueue
	name  string
	done  chan struct{}
	once  sync.Once
}

func (c *testConsumer) Receive(ctx context.Context) (Delivery, error) {
	for {
		task, notify, wait := c.queue.next(c.name)
		if task != nil {
			return &testDelivery{queue: c.queue, task: task}, nil
		}

		var timer *time.Timer
		var timerC <-chan time.Time
		if wait > 0 {
			timer = time.NewTimer(wait)
			timerC = timer.C
		}

		select {
		case <-ctx.Done():
			stopTestTimer(timer)
			return nil, ctx.Err()
		case <-c.done:
			stopTestTimer(timer)
			return nil, ErrConsumerClosed
		case <-notify:
			stopTestTimer(timer)
		case <-timerC:
		}
	}
}

func (c *testConsumer) Close() error {
	c.once.Do(func() {
		close(c.done)
	})
	return nil
}

func (q *testQueue) next(queueName string) (*Task, <-chan struct{}, time.Duration) {
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
		task = task.clone()
		task.Attempt++
		return task, q.notify, 0
	}

	return nil, q.notify, wait
}

type testDelivery struct {
	queue *testQueue
	task  *Task
}

func (d *testDelivery) Task() *Task {
	if d == nil {
		return nil
	}
	return d.task
}

func (*testDelivery) Ack(ctx context.Context) error {
	return ctx.Err()
}

func (d *testDelivery) Retry(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.clone()
	task.AvailableAt = time.Now().UTC().Add(delay)
	return d.queue.Enqueue(ctx, task)
}

func (d *testDelivery) DeadLetter(ctx context.Context, reason string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d == nil || d.task == nil {
		return nil
	}

	task := d.task.clone()
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata["queue.dead_letter.reason"] = reason

	d.queue.mu.Lock()
	defer d.queue.mu.Unlock()

	d.queue.deadLetter[task.Queue] = append(d.queue.deadLetter[task.Queue], task)
	return nil
}

func (q *testQueue) DeadLetters(queueName string) []*Task {
	if queueName == "" {
		queueName = DefaultQueue
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	tasks := q.deadLetter[queueName]
	cloned := make([]*Task, 0, len(tasks))
	for _, task := range tasks {
		cloned = append(cloned, task.clone())
	}
	return cloned
}

func (q *testQueue) signal() {
	close(q.notify)
	q.notify = make(chan struct{})
}

func stopTestTimer(timer *time.Timer) {
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
