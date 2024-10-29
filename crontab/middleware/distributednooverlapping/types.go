package distributednooverlapping

import (
	"context"
	"time"

	"github.com/flc1125/go-cron/v4"
)

type Mutex interface {
	// Lock tries to acquire the mutex for the job.
	// If the mutex is acquired, it returns true.
	// If the mutex is not acquired, it returns false.
	Lock(ctx context.Context, job JobWithMutex) (bool, error)

	// Unlock releases the mutex for the job.
	Unlock(ctx context.Context, job JobWithMutex) error
}

type JobWithMutex interface {
	cron.Job

	// GetMutexKey returns the key of the mutex.
	GetMutexKey() string

	// GetMutexTTL returns the ttl of the mutex.
	// The ttl suggests greater than the running time of the job.
	GetMutexTTL() time.Duration
}

// DefaultTTLJobWithMutex is a default implementation of JobWithMutex.
type DefaultTTLJobWithMutex struct{}

func (DefaultTTLJobWithMutex) GetMutexTTL() time.Duration {
	return time.Hour
}
