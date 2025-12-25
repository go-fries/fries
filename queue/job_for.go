package queue

import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

// ErrPayloadTypeMismatch is returned when the payload type does not match
var ErrPayloadTypeMismatch = errors.New("queue: payload type mismatch")

// JobFor is a type-safe wrapper around Job
type JobFor[T any] struct {
	job Job
}

// NewJobFor creates a new type-safe job with the given payload
func NewJobFor[T any](payload T, opts ...JobOption) *JobFor[T] {
	j := NewJob(payload, opts...)
	// Set payloadType on the underlying job
	if jImpl, ok := j.(*job); ok {
		jImpl.payloadType = typeNameOf[T]()
	}
	return &JobFor[T]{job: j}
}

// JobAs converts a Job to *JobFor[T], returns error if type mismatch
func JobAs[T any](j Job) (*JobFor[T], error) {
	expectedType := typeNameOf[T]()

	// If payloadType is set, check it matches
	if j.PayloadType() != "" && j.PayloadType() != expectedType {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrPayloadTypeMismatch, expectedType, j.PayloadType())
	}

	// Try type assertion
	if _, ok := j.Payload().(T); !ok {
		return nil, fmt.Errorf("%w: cannot convert payload to %s", ErrPayloadTypeMismatch, expectedType)
	}

	return &JobFor[T]{job: j}, nil
}

// MustJobAs converts a Job to *JobFor[T], panics if type mismatch
func MustJobAs[T any](j Job) *JobFor[T] {
	jobFor, err := JobAs[T](j)
	if err != nil {
		panic(err)
	}
	return jobFor
}

// Job returns the underlying Job
func (j *JobFor[T]) Job() Job {
	return j.job
}

// Payload returns the typed payload
func (j *JobFor[T]) Payload() (T, error) {
	payload, ok := j.job.Payload().(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("%w: cannot convert payload to %s", ErrPayloadTypeMismatch, typeNameOf[T]())
	}
	return payload, nil
}

// MustPayload returns the typed payload, panics if type mismatch
func (j *JobFor[T]) MustPayload() T {
	payload, err := j.Payload()
	if err != nil {
		panic(err)
	}
	return payload
}

// ID returns the job's unique identifier
func (j *JobFor[T]) ID() string {
	return j.job.ID()
}

// Queue returns the queue name
func (j *JobFor[T]) Queue() string {
	return j.job.Queue()
}

// Priority returns the job's priority
func (j *JobFor[T]) Priority() int {
	return j.job.Priority()
}

// Delay returns the job's delay duration
func (j *JobFor[T]) Delay() time.Duration {
	return j.job.Delay()
}

// MaxAttempts returns the maximum number of retry attempts
func (j *JobFor[T]) MaxAttempts() int {
	return j.job.MaxAttempts()
}

// Attempts returns the current attempt count
func (j *JobFor[T]) Attempts() int {
	return j.job.Attempts()
}

// AvailableAt returns when the job becomes available
func (j *JobFor[T]) AvailableAt() time.Time {
	return j.job.AvailableAt()
}

// CreatedAt returns the job creation time
func (j *JobFor[T]) CreatedAt() time.Time {
	return j.job.CreatedAt()
}

// FailedAt returns the last failure time
func (j *JobFor[T]) FailedAt() *time.Time {
	return j.job.FailedAt()
}

// LastError returns the last error message
func (j *JobFor[T]) LastError() string {
	return j.job.LastError()
}

// PayloadType returns the payload type name
func (j *JobFor[T]) PayloadType() string {
	return j.job.PayloadType()
}

// typeNameOf returns the type name of T
func typeNameOf[T any]() string {
	var t T
	return reflect.TypeOf(t).String()
}
