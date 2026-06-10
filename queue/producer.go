package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"maps"
	"time"

	"github.com/go-fries/fries/codec/v3"
)

type enqueueConfig struct {
	id       string
	queue    string
	metadata map[string]string
	delay    time.Duration
}

type producerConfig struct {
	observer Observer
}

// EnqueueOption is an option that configures how a task is enqueued.
type EnqueueOption interface {
	applyEnqueue(*enqueueConfig)
}

type enqueueOptionFunc func(*enqueueConfig)

func (f enqueueOptionFunc) applyEnqueue(c *enqueueConfig) {
	f(c)
}

// WithID sets the task ID.
func WithID(id string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		c.id = id
	})
}

// WithMetadataValue adds a single task metadata value.
func WithMetadataValue(key, value string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if c.metadata == nil {
			c.metadata = make(map[string]string)
		}
		c.metadata[key] = value
	})
}

// WithMetadata adds task metadata values.
func WithMetadata(metadata map[string]string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if len(metadata) == 0 {
			return
		}
		if c.metadata == nil {
			c.metadata = make(map[string]string, len(metadata))
		}
		maps.Copy(c.metadata, metadata)
	})
}

// WithDelay delays task availability.
func WithDelay(delay time.Duration) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if delay > 0 {
			c.delay = delay
		}
	})
}

func newEnqueueConfig(opts ...EnqueueOption) *enqueueConfig {
	c := &enqueueConfig{
		queue: DefaultQueue,
	}
	for _, opt := range opts {
		opt.applyEnqueue(c)
	}
	if c.id == "" {
		c.id = newID()
	}
	return c
}

// ProducerOption is an option that configures a Producer.
type ProducerOption interface {
	applyProducer(*producerConfig)
}

func newProducerConfig(opts ...ProducerOption) *producerConfig {
	c := &producerConfig{}
	for _, opt := range opts {
		opt.applyProducer(c)
	}
	return c
}

// Producer creates tasks in a queue.
type Producer struct {
	queue    Queue
	observer Observer
}

// NewProducer creates a producer that writes to q.
func NewProducer(q Queue, opts ...ProducerOption) *Producer {
	c := newProducerConfig(opts...)
	return &Producer{
		queue:    q,
		observer: c.observer,
	}
}

// Enqueue creates a task with a byte payload and stores it in the queue.
func (p *Producer) Enqueue(ctx context.Context, taskType string, payload []byte, opts ...EnqueueOption) (*Task, error) {
	if taskType == "" {
		return nil, ErrInvalidTaskType
	}

	c := newEnqueueConfig(opts...)
	now := time.Now().UTC()
	task := &Task{
		ID:          c.id,
		Type:        taskType,
		Queue:       c.queue,
		Payload:     append([]byte(nil), payload...),
		Metadata:    c.metadata,
		CreatedAt:   now,
		AvailableAt: now.Add(c.delay),
	}
	observeCtx := p.observe(ctx, Event{
		Kind:  EventEnqueueStarted,
		Queue: task.Queue,
		Task:  taskInfo(task),
		Delay: c.delay,
	})
	if err := p.queue.Enqueue(observeCtx, task); err != nil {
		p.observe(observeCtx, Event{
			Kind:  EventEnqueueFailed,
			Queue: task.Queue,
			Task:  taskInfo(task),
			Delay: c.delay,
			Err:   err,
		})
		return nil, err
	}
	p.observe(observeCtx, Event{
		Kind:  EventEnqueued,
		Queue: task.Queue,
		Task:  taskInfo(task),
		Delay: c.delay,
	})
	return task.clone(), nil
}

func (p *Producer) observe(ctx context.Context, event Event) context.Context {
	if p == nil || p.observer == nil {
		return ctx
	}
	observedCtx := p.observer.ObserveQueue(ctx, event)
	if observedCtx == nil {
		return ctx
	}
	return observedCtx
}

// TaskProducer enqueues one task type with a structured payload.
type TaskProducer[T any] struct {
	producer *Producer
	taskType string
}

// NewTaskProducer creates a typed producer bound to taskType.
func NewTaskProducer[T any](producer *Producer, taskType string) *TaskProducer[T] {
	return &TaskProducer[T]{
		producer: producer,
		taskType: taskType,
	}
}

// Enqueue encodes payload with the default JSON codec and enqueues it as the producer task type.
func (p *TaskProducer[T]) Enqueue(ctx context.Context, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueFor(ctx, p.producer, p.taskType, payload, opts...)
}

// EnqueueFor encodes payload with the default JSON codec and enqueues it as taskType.
func EnqueueFor[T any](ctx context.Context, producer *Producer, taskType string, payload T, opts ...EnqueueOption) (*Task, error) {
	return EnqueueForWithCodec(ctx, producer, taskType, payload, defaultCodec, opts...)
}

// EnqueueForWithCodec encodes payload with codec and enqueues it as taskType.
func EnqueueForWithCodec[T any](
	ctx context.Context,
	producer *Producer,
	taskType string,
	payload T,
	codec codec.Codec,
	opts ...EnqueueOption,
) (*Task, error) {
	if producer == nil {
		return nil, errors.New("queue: producer is nil")
	}
	if codec == nil {
		codec = defaultCodec
	}

	data, err := codec.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return producer.Enqueue(ctx, taskType, data, opts...)
}

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(data[:])
}
