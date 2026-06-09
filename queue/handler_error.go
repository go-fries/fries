package queue

import (
	"errors"
	"fmt"
	"time"
)

const defaultDeadLetterReason = "queue: dead letter requested"

// ErrDiscard tells a Worker to acknowledge the task without retrying or
// dead-lettering it.
var ErrDiscard = errors.New("queue: discard task")

type retryAfterError struct {
	delay time.Duration
}

// RetryAfter tells a Worker to retry the task after delay.
//
// The retry still respects the Worker's RetryPolicy retry budget. When the
// retry policy says no retry is available, the task is dead-lettered.
func RetryAfter(delay time.Duration) error {
	if delay < 0 {
		delay = 0
	}
	return &retryAfterError{delay: delay}
}

func (e *retryAfterError) Error() string {
	return fmt.Sprintf("queue: retry after %s", e.delay)
}

func retryAfterDelay(err error) (time.Duration, bool) {
	var target *retryAfterError
	if !errors.As(err, &target) {
		return 0, false
	}
	return target.delay, true
}

type deadLetterError struct {
	reason string
}

// DeadLetter tells a Worker to move the task to dead-letter storage immediately.
func DeadLetter(reason string) error {
	if reason == "" {
		reason = defaultDeadLetterReason
	}
	return &deadLetterError{reason: reason}
}

func (e *deadLetterError) Error() string {
	return fmt.Sprintf("queue: dead letter: %s", e.reason)
}

func deadLetterReason(err error) (string, bool) {
	var target *deadLetterError
	if !errors.As(err, &target) {
		return "", false
	}
	return target.reason, true
}
