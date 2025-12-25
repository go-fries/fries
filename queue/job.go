package queue

import (
	"time"

	"github.com/google/uuid"
)

// Job represents a queue task
type Job struct {
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

// NewJob creates a new job with the given payload and options
func NewJob(payload any, opts ...JobOption) *Job {
	now := time.Now()
	job := &Job{
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
		opt(job)
	}

	// Apply delay to availableAt if set
	if job.delay > 0 {
		job.availableAt = job.createdAt.Add(job.delay)
	}

	return job
}

// JobOption configures a Job
type JobOption func(*Job)

// WithJobID sets a custom job ID
func WithJobID(id string) JobOption {
	return func(j *Job) {
		j.id = id
	}
}

// WithQueue sets the queue name
func WithQueue(name string) JobOption {
	return func(j *Job) {
		j.queue = name
	}
}

// WithPriority sets the job priority (higher values = higher priority)
func WithPriority(p int) JobOption {
	return func(j *Job) {
		j.priority = p
	}
}

// WithDelay sets the delay before the job becomes available
func WithDelay(d time.Duration) JobOption {
	return func(j *Job) {
		j.delay = d
	}
}

// WithMaxAttempts sets the maximum number of retry attempts
func WithMaxAttempts(n int) JobOption {
	return func(j *Job) {
		if n < 1 {
			n = 1
		}
		j.maxAttempts = n
	}
}

// ID returns the job's unique identifier
func (j *Job) ID() string {
	return j.id
}

// Payload returns the job's payload
func (j *Job) Payload() any {
	return j.payload
}

// PayloadType returns the payload type name
func (j *Job) PayloadType() string {
	return j.payloadType
}

// Queue returns the queue name
func (j *Job) Queue() string {
	return j.queue
}

// Priority returns the job's priority
func (j *Job) Priority() int {
	return j.priority
}

// Delay returns the job's delay duration
func (j *Job) Delay() time.Duration {
	return j.delay
}

// MaxAttempts returns the maximum number of retry attempts
func (j *Job) MaxAttempts() int {
	return j.maxAttempts
}

// Attempts returns the current attempt count
func (j *Job) Attempts() int {
	return j.attempts
}

// AvailableAt returns when the job becomes available
func (j *Job) AvailableAt() time.Time {
	return j.availableAt
}

// CreatedAt returns the job creation time
func (j *Job) CreatedAt() time.Time {
	return j.createdAt
}

// FailedAt returns the last failure time (nil if never failed)
func (j *Job) FailedAt() *time.Time {
	return j.failedAt
}

// LastError returns the last error message
func (j *Job) LastError() string {
	return j.lastError
}

// IncrAttempts increments the attempt count (for driver use)
func (j *Job) IncrAttempts() {
	j.attempts++
}

// SetAvailableAt sets when the job becomes available (for driver use)
func (j *Job) SetAvailableAt(t time.Time) {
	j.availableAt = t
}

// SetFailed marks the job as failed with the given error (for driver use)
func (j *Job) SetFailed(err error) {
	now := time.Now()
	j.failedAt = &now
	if err != nil {
		j.lastError = err.Error()
	}
}
