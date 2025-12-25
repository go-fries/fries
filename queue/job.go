package queue

import (
	"time"

	"github.com/google/uuid"
)

// Job defines the interface for a queue job
type Job interface {
	// Read-only accessors
	ID() string
	Payload() any
	PayloadType() string
	Queue() string
	Priority() int
	Delay() time.Duration
	MaxAttempts() int
	Attempts() int
	AvailableAt() time.Time
	CreatedAt() time.Time
	FailedAt() *time.Time
	LastError() string

	// Mutable methods (for driver use)
	IncrAttempts()
	SetAvailableAt(t time.Time)
	SetFailed(err error)
}

// job is the default implementation of Job interface
type job struct {
	id          string        // unique job identifier
	payload     any           // job payload (serialized by codec)
	payloadType string        // payload type name (for serialization)
	queue       string        // queue name
	priority    int           // priority (higher is more important, default 0)
	delay       time.Duration // delay before execution
	maxAttempts int           // maximum retry attempts (default 1, no retry)
	attempts    int           // current attempt count
	availableAt time.Time     // when the job becomes available
	createdAt   time.Time     // job creation time
	failedAt    *time.Time    // last failure time
	lastError   string        // last error message
}

// Ensure job implements Job interface
var _ Job = (*job)(nil)

// NewJob creates a new job with the given payload and options
func NewJob(payload any, opts ...JobOption) Job {
	now := time.Now()
	j := &job{
		id:          uuid.New().String(),
		payload:     payload,
		queue:       "default",
		priority:    0,
		delay:       0,
		maxAttempts: 1,
		attempts:    0,
		availableAt: now,
		createdAt:   now,
	}

	for _, opt := range opts {
		opt(j)
	}

	// Apply delay to availableAt if set
	if j.delay > 0 {
		j.availableAt = j.createdAt.Add(j.delay)
	}

	return j
}

// JobOption configures a job
type JobOption func(*job)

// WithJobID sets a custom job ID
func WithJobID(id string) JobOption {
	return func(j *job) {
		j.id = id
	}
}

// WithQueue sets the queue name
func WithQueue(name string) JobOption {
	return func(j *job) {
		j.queue = name
	}
}

// WithPriority sets the job priority (higher values = higher priority)
func WithPriority(p int) JobOption {
	return func(j *job) {
		j.priority = p
	}
}

// WithDelay sets the delay before the job becomes available
func WithDelay(d time.Duration) JobOption {
	return func(j *job) {
		j.delay = d
	}
}

// WithMaxAttempts sets the maximum number of retry attempts
func WithMaxAttempts(n int) JobOption {
	return func(j *job) {
		if n < 1 {
			n = 1
		}
		j.maxAttempts = n
	}
}

// ID returns the job's unique identifier
func (j *job) ID() string {
	return j.id
}

// Payload returns the job's payload
func (j *job) Payload() any {
	return j.payload
}

// PayloadType returns the payload type name
func (j *job) PayloadType() string {
	return j.payloadType
}

// Queue returns the queue name
func (j *job) Queue() string {
	return j.queue
}

// Priority returns the job's priority
func (j *job) Priority() int {
	return j.priority
}

// Delay returns the job's delay duration
func (j *job) Delay() time.Duration {
	return j.delay
}

// MaxAttempts returns the maximum number of retry attempts
func (j *job) MaxAttempts() int {
	return j.maxAttempts
}

// Attempts returns the current attempt count
func (j *job) Attempts() int {
	return j.attempts
}

// AvailableAt returns when the job becomes available
func (j *job) AvailableAt() time.Time {
	return j.availableAt
}

// CreatedAt returns the job creation time
func (j *job) CreatedAt() time.Time {
	return j.createdAt
}

// FailedAt returns the last failure time (nil if never failed)
func (j *job) FailedAt() *time.Time {
	return j.failedAt
}

// LastError returns the last error message
func (j *job) LastError() string {
	return j.lastError
}

// IncrAttempts increments the attempt count (for driver use)
func (j *job) IncrAttempts() {
	j.attempts++
}

// SetAvailableAt sets when the job becomes available (for driver use)
func (j *job) SetAvailableAt(t time.Time) {
	j.availableAt = t
}

// SetFailed marks the job as failed with the given error (for driver use)
func (j *job) SetFailed(err error) {
	now := time.Now()
	j.failedAt = &now
	if err != nil {
		j.lastError = err.Error()
	}
}
