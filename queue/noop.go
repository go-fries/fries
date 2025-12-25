package queue

import "context"

// NoopDriver is a no-op implementation of the Driver interface
// Useful for testing or disabling the queue
type NoopDriver struct{}

var _ Driver = (*NoopDriver)(nil)

// Push does nothing and returns nil
func (NoopDriver) Push(context.Context, *Job) error {
	return nil
}

// Pop blocks until the context is cancelled
func (NoopDriver) Pop(ctx context.Context, _ ...string) (*Job, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// Ack does nothing and returns nil
func (NoopDriver) Ack(context.Context, *Job) error {
	return nil
}

// Fail does nothing and returns nil
func (NoopDriver) Fail(context.Context, *Job, error) error {
	return nil
}

// Size returns 0
func (NoopDriver) Size(context.Context, string) (int64, error) {
	return 0, nil
}

// Dead returns 0
func (NoopDriver) Dead(context.Context, string) (int64, error) {
	return 0, nil
}

// Clear does nothing and returns nil
func (NoopDriver) Clear(context.Context, string) error {
	return nil
}

// ClearDead does nothing and returns nil
func (NoopDriver) ClearDead(context.Context, string) error {
	return nil
}
