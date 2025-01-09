package snap

import (
	"sync"
	"time"
)

type Snap[T any] struct {
	value T
	mu    sync.RWMutex

	refresh     func() (T, error)
	expired     time.Time     // expiration time
	interval    time.Duration // refresh interval
	async       bool
	isRefreshed bool // is refreshed
}

type Option[T any] func(*Snap[T])

func Interval[T any](interval time.Duration) Option[T] {
	return func(s *Snap[T]) {
		s.interval = interval
	}
}

func Async[T any](async bool) Option[T] {
	return func(s *Snap[T]) {
		s.async = async
	}
}

func New[T any](refresh func() (T, error), opts ...Option[T]) *Snap[T] {
	s := &Snap[T]{
		refresh:  refresh,
		interval: time.Minute,
		async:    true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Snap[T]) Get() T {
	s.attemptRefresh()

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *Snap[T]) Refresh() error {
	value, err := s.refresh()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = value
	s.expired = time.Now().Add(s.interval)
	s.isRefreshed = true
	return nil
}

func (s *Snap[T]) attemptRefresh() {
	if !s.canRefresh() {
		return
	}

	if s.async && !s.isRefreshed { // if not refreshed, the first must be synchronous
		go s.Refresh() // nolint:errcheck
	} else {
		_ = s.Refresh()
	}
}

func (s *Snap[T]) canRefresh() bool {
	return s.expired.IsZero() || time.Now().After(s.expired)
}
