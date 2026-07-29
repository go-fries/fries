package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/ratelimit/v4"
)

type record struct {
	tat int64
}

// Store keeps rate-limit state in process memory.
//
// A Store is safe for concurrent use. Expired keys are removed
// opportunistically without starting a background goroutine.
type Store struct {
	mu          sync.Mutex
	records     map[string]record
	now         func() time.Time
	nextCleanup int64
}

var _ ratelimit.Store = (*Store)(nil)

// New creates an empty [Store].
func New() *Store {
	return &Store{
		records: make(map[string]record),
		now:     time.Now,
	}
}

// Take atomically decides whether request.Cost units can be consumed.
func (s *Store) Take(
	ctx context.Context,
	request ratelimit.TakeRequest,
) (ratelimit.Decision, error) {
	if err := validateContext(ctx); err != nil {
		return ratelimit.Decision{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := context.Cause(ctx); err != nil {
		return ratelimit.Decision{}, err
	}

	now := s.now().UnixMicro()
	s.cleanup(now)

	interval := emissionInterval(request.Limit)
	burstOffset := interval * int64(request.Limit.Burst)
	increment := interval * int64(request.Cost)
	tat := now
	if current, exists := s.records[request.Key]; exists && current.tat > tat {
		tat = current.tat
	}

	newTAT := tat + increment
	if newTAT > now+burstOffset {
		return ratelimit.Decision{
			Limit:      request.Limit,
			Allowed:    false,
			Remaining:  remaining(now, tat, interval, burstOffset),
			RetryAfter: microseconds(newTAT - burstOffset - now),
			ResetAfter: microseconds(tat - now),
		}, nil
	}

	s.records[request.Key] = record{tat: newTAT}
	s.scheduleCleanup(newTAT)
	return ratelimit.Decision{
		Limit:      request.Limit,
		Allowed:    true,
		Remaining:  remaining(now, newTAT, interval, burstOffset),
		ResetAfter: microseconds(newTAT - now),
	}, nil
}

// Reset removes the stored state for key.
func (s *Store) Reset(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := context.Cause(ctx); err != nil {
		return err
	}

	s.cleanup(s.now().UnixMicro())
	delete(s.records, key)
	return nil
}

func (s *Store) cleanup(now int64) {
	if s.nextCleanup == 0 || now < s.nextCleanup {
		return
	}

	s.nextCleanup = 0
	for key, current := range s.records {
		if current.tat <= now {
			delete(s.records, key)
			continue
		}
		s.scheduleCleanup(current.tat)
	}
}

func (s *Store) scheduleCleanup(at int64) {
	if s.nextCleanup == 0 || at < s.nextCleanup {
		s.nextCleanup = at
	}
}

func emissionInterval(limit ratelimit.Limit) int64 {
	period := int64(limit.Period)
	rate := int64(limit.Rate)
	interval := period / rate
	if period%rate != 0 {
		interval++
	}
	microsecond := int64(time.Microsecond)
	result := interval / microsecond
	if interval%microsecond != 0 {
		result++
	}
	return result
}

func remaining(now, tat, interval, burstOffset int64) int {
	available := now + burstOffset - tat
	if available <= 0 {
		return 0
	}
	return int(available / interval)
}

func microseconds(value int64) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Microsecond
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ratelimit.ErrInvalidContext
	}
	return context.Cause(ctx)
}
