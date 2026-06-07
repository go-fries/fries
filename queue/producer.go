package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"maps"
	"time"
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
	id := c.id
	if id == "" {
		id = newID()
	}

	now := time.Now().UTC()
	task := &Task{
		ID:          id,
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

func newID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(data[:])
}
