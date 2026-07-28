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
type Store struct {
	mu      sync.Mutex
	records map[string]record
	now     func() time.Time
}

var _ idempotency.Store = (*Store)(nil)

// New creates an empty Store.
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
	current, exists := s.records[request.Key]
	if exists && !now.Before(current.expiresAt) {
		delete(s.records, request.Key)
		exists = false
	}
	if !exists {
		s.records[request.Key] = record{
			status:      idempotency.BeginInProgress,
			token:       request.Token,
			fingerprint: request.Fingerprint,
			expiresAt:   now.Add(request.TTL),
		}
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
	current, exists := s.current(request.Key, now)
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

	current, exists := s.current(request.Key, s.now())
	if !exists ||
		current.status != idempotency.BeginInProgress ||
		current.token != request.Token {
		return idempotency.ErrClaimLost
	}
	delete(s.records, request.Key)
	return nil
}

func (s *Store) current(key string, now time.Time) (record, bool) {
	current, exists := s.records[key]
	if exists && !now.Before(current.expiresAt) {
		delete(s.records, key)
		return record{}, false
	}
	return current, exists
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
