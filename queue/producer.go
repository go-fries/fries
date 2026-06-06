package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"maps"
	"time"
)

type enqueueConfig struct {
	id             string
	queue          string
	headers        map[string]string
	idempotencyKey string
	delay          time.Duration
	now            func() time.Time
}

type EnqueueOption interface {
	apply(*enqueueConfig)
}

type enqueueOptionFunc func(*enqueueConfig)

func (f enqueueOptionFunc) apply(c *enqueueConfig) {
	f(c)
}

func WithID(id string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		c.id = id
	})
}

func WithQueue(name string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if name != "" {
			c.queue = name
		}
	})
}

func WithHeader(key, value string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if c.headers == nil {
			c.headers = make(map[string]string)
		}
		c.headers[key] = value
	})
}

func WithHeaders(headers map[string]string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		if len(headers) == 0 {
			return
		}
		if c.headers == nil {
			c.headers = make(map[string]string, len(headers))
		}
		maps.Copy(c.headers, headers)
	})
}

func WithIdempotencyKey(key string) EnqueueOption {
	return enqueueOptionFunc(func(c *enqueueConfig) {
		c.idempotencyKey = key
	})
}

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
		now:   func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

type Producer struct {
	backend Backend
}

func NewProducer(backend Backend) *Producer {
	return &Producer{backend: backend}
}

func (p *Producer) Enqueue(ctx context.Context, taskType string, payload []byte, opts ...EnqueueOption) (*Task, error) {
	if taskType == "" {
		return nil, ErrInvalidTaskType
	}

	c := newEnqueueConfig(opts...)
	id := c.id
	if id == "" {
		id = newID()
	}

	now := c.now()
	task := &Task{
		ID:             id,
		Type:           taskType,
		Queue:          c.queue,
		Payload:        append([]byte(nil), payload...),
		Headers:        c.headers,
		IdempotencyKey: c.idempotencyKey,
		CreatedAt:      now,
		AvailableAt:    now.Add(c.delay),
	}
	if err := p.backend.Enqueue(ctx, task); err != nil {
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
