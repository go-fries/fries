package memory

import (
	"context"
	"sync"
	"time"

	"github.com/go-fries/fries/idempotency/v4"
)

type record struct {
	status      idempotency.BeginStatus
	token       string
	fingerprint string
	data        []byte
	expiresAt   time.Time
}

// Store keeps idempotency records in process memory.
// A Store is safe for concurrent use.
type Store struct {
	mu          sync.Mutex
	records     map[string]record
	now         func() time.Time
	nextCleanup time.Time
}

var _ idempotency.Store = (*Store)(nil)

// New creates an empty [Store].
func New() *Store {
	return &Store{
		records: make(map[string]record),
		now:     time.Now,
	}
}

// Begin atomically creates a claim or reports the current state of the key.
func (s *Store) Begin(
	ctx context.Context,
	request idempotency.BeginRequest,
) (idempotency.BeginResult, error) {
	if err := validateContext(ctx); err != nil {
		return idempotency.BeginResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanup(now)
	current, exists := s.records[request.Key]
	if !exists {
		current = record{
			status:      idempotency.BeginInProgress,
			token:       request.Token,
			fingerprint: request.Fingerprint,
			expiresAt:   now.Add(request.TTL),
		}
		s.records[request.Key] = current
		s.scheduleCleanup(current.expiresAt)
		return idempotency.BeginResult{Status: idempotency.BeginAcquired}, nil
	}
	if request.Fingerprint != "" && request.Fingerprint != current.fingerprint {
		return idempotency.BeginResult{}, idempotency.ErrKeyConflict
	}
	return idempotency.BeginResult{
		Status: current.status,
		Result: clone(current.data),
	}, nil
}

// Complete atomically replaces an owned active claim with a completed record.
func (s *Store) Complete(
	ctx context.Context,
	request idempotency.CompleteRequest,
) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	s.cleanup(now)
	current, exists := s.records[request.Key]
	if !exists ||
		current.status != idempotency.BeginInProgress ||
		current.token != request.Token {
		return idempotency.ErrClaimLost
	}
	current.status = idempotency.BeginCompleted
	current.token = ""
	current.data = clone(request.Result)
	current.expiresAt = now.Add(request.TTL)
	s.records[request.Key] = current
	s.scheduleCleanup(current.expiresAt)
	return nil
}

// Abort atomically removes an owned active claim.
func (s *Store) Abort(
	ctx context.Context,
	request idempotency.AbortRequest,
) error {
	if err := validateContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanup(s.now())
	current, exists := s.records[request.Key]
	if !exists ||
		current.status != idempotency.BeginInProgress ||
		current.token != request.Token {
		return idempotency.ErrClaimLost
	}
	delete(s.records, request.Key)
	return nil
}

func (s *Store) cleanup(now time.Time) {
	if s.nextCleanup.IsZero() || now.Before(s.nextCleanup) {
		return
	}

	s.nextCleanup = time.Time{}
	for key, current := range s.records {
		if !now.Before(current.expiresAt) {
			delete(s.records, key)
			continue
		}
		s.scheduleCleanup(current.expiresAt)
	}
}

func (s *Store) scheduleCleanup(expiresAt time.Time) {
	if s.nextCleanup.IsZero() || expiresAt.Before(s.nextCleanup) {
		s.nextCleanup = expiresAt
	}
}

func clone(data []byte) []byte {
	return append([]byte(nil), data...)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return idempotency.ErrInvalidContext
	}
	return context.Cause(ctx)
}
