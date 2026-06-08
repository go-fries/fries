package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// EnqueueOption configures task enqueue behavior.
type EnqueueOption interface {
	apply(*enqueueConfig)
}

type enqueueOptionFunc func(*enqueueConfig)

func (f enqueueOptionFunc) apply(c *enqueueConfig) {
	f(c)
}

// WithID sets the task ID.
func WithID(id string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		c.id = id
	})
}

// WithQueue sets the queue name for a task.
func WithQueue(name string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if name != "" {
			c.queue = name
		}
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
		opt.apply(c)
	}
	if c.id == "" {
		c.id = newID()
	}
	return c
}

// Producer creates tasks in a queue.
type Producer struct {
	queue Queue
}

// NewProducer creates a producer that writes to q.
func NewProducer(q Queue) *Producer {
	return &Producer{queue: q}
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
	if err := p.queue.Enqueue(ctx, task); err != nil {
		return nil, err
	}
	return task.clone(), nil
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
