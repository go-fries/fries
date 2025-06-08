package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Job interface {
	ID() string

	Queue() string

	Subject() string

	Meta() map[string][]string

	Data() []byte

	Release(ctx context.Context) error

	Later(ctx context.Context, delay time.Duration) error

	LaterOn(ctx context.Context, delay time.Time) error
}

type job struct {
	id         string
	queue      string
	subject    string
	data       []byte
	sync       bool
	maxRetries int
	meta       map[string][]string
}

type JobOption interface {
	apply(*job)
}

type jobOptionFunc func(*job)

func (f jobOptionFunc) apply(j *job) {
	f(j)
}

func WithID(id string) JobOption {
	return jobOptionFunc(func(j *job) {
		j.id = id
	})
}

var _ Job = (*job)(nil)

func newJob(subject string, data []byte, opts ...JobOption) *job {
	j := &job{
		id:         uuid.New().String(),
		queue:      "default", // Default queue, can be set via options
		subject:    subject,
		data:       data,
		sync:       false, // Default value, can be set via options
		maxRetries: 3,     // Default value, can be set via options
		meta:       make(map[string][]string),
	}

	for _, opt := range opts {
		opt.apply(j)
	}

	return j
}

func (j *job) toMessage() *Message {
	return &Message{
		ID:         j.ID(),
		Queue:      j.Queue(),
		Meta:       j.Meta(),
		Subject:    j.Subject(),
		Data:       j.Data(),
		MaxRetries: 0, // Default value, can be set via options
		Retries:    0, // Default value, can be set via options
	}
}

func (j *job) ID() string {
	return j.id
}

func (j *job) Queue() string {
	return j.queue
}

func (j *job) Subject() string {
	return j.subject
}

func (j *job) Meta() map[string][]string {
	return j.meta
}

func (j *job) Data() []byte {
	return j.data
}

func (j *job) Release(ctx context.Context) error {
	// TODO implement me
	panic("implement me")
}

func (j *job) Later(ctx context.Context, delay time.Duration) error {
	// TODO implement me
	panic("implement me")
}

func (j *job) LaterOn(ctx context.Context, delay time.Time) error {
	// TODO implement me
	panic("implement me")
}

func parseMessage(msg Message) (*job, error) {
	return &job{
		id:         msg.ID,
		queue:      msg.Queue,
		subject:    msg.Subject,
		data:       msg.Data,
		maxRetries: msg.MaxRetries,
		meta:       msg.Meta,
	}, nil
}
