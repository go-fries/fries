package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-fries/fries/idempotency/v4"
	goredis "github.com/redis/go-redis/v9"
)

const (
	beginAcquiredCode int64 = iota + 1
	beginInProgressCode
	beginCompletedCode
	beginConflictCode
)

var (
	beginScript = goredis.NewScript(`
local status = redis.call("hget", KEYS[1], "status")
if not status then
    redis.call(
        "hset",
        KEYS[1],
        "status", "in_progress",
        "token", ARGV[1],
        "fingerprint", ARGV[2]
    )
    redis.call("pexpire", KEYS[1], ARGV[3])
    return {1, ""}
end

local fingerprint = redis.call("hget", KEYS[1], "fingerprint") or ""
if ARGV[2] ~= "" and fingerprint ~= ARGV[2] then
    return {4, ""}
end
if status == "in_progress" then
    return {2, ""}
end
if status == "completed" then
    return {3, redis.call("hget", KEYS[1], "result") or ""}
end
return {0, ""}
`)
	completeScript = goredis.NewScript(`
if redis.call("hget", KEYS[1], "status") ~= "in_progress" then
    return 0
end
if redis.call("hget", KEYS[1], "token") ~= ARGV[1] then
    return 0
end
redis.call(
    "hset",
    KEYS[1],
    "status", "completed",
    "result", ARGV[2]
)
redis.call("hdel", KEYS[1], "token")
redis.call("pexpire", KEYS[1], ARGV[3])
return 1
`)
	abortScript = goredis.NewScript(`
if redis.call("hget", KEYS[1], "status") ~= "in_progress" then
    return 0
end
if redis.call("hget", KEYS[1], "token") ~= ARGV[1] then
    return 0
end
return redis.call("del", KEYS[1])
`)
)

// Store keeps idempotency records in Redis.
// A Store is safe for concurrent use.
type Store struct {
	client goredis.UniversalClient
	prefix string
}

var _ idempotency.Store = (*Store)(nil)

// New creates a Redis-backed [Store].
//
// New panics if client is nil.
func New(client goredis.UniversalClient, options ...Option) *Store {
	if client == nil {
		panic("idempotency/redis: nil client")
	}
	c := newConfig(options...)
	return &Store{
		client: client,
		prefix: c.prefix,
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

	key := s.prefix + request.Key
	raw, err := beginScript.Run(
		ctx,
		s.client,
		[]string{key},
		request.Token,
		request.Fingerprint,
		milliseconds(request.TTL),
	).Result()
	if err != nil {
		return idempotency.BeginResult{}, fmt.Errorf(
			"idempotency/redis: begin %q: %w",
			key,
			err,
		)
	}

	result, ok := raw.([]any)
	if !ok || len(result) != 2 {
		return idempotency.BeginResult{}, fmt.Errorf(
			"idempotency/redis: begin %q returned %T",
			key,
			raw,
		)
	}
	code, ok := result[0].(int64)
	if !ok {
		return idempotency.BeginResult{}, fmt.Errorf(
			"idempotency/redis: begin %q returned status %T",
			key,
			result[0],
		)
	}
	switch code {
	case beginAcquiredCode:
		return idempotency.BeginResult{Status: idempotency.BeginAcquired}, nil
	case beginInProgressCode:
		return idempotency.BeginResult{Status: idempotency.BeginInProgress}, nil
	case beginCompletedCode:
		data, err := resultBytes(result[1])
		if err != nil {
			return idempotency.BeginResult{}, fmt.Errorf(
				"idempotency/redis: begin %q: %w",
				key,
				err,
			)
		}
		return idempotency.BeginResult{
			Status: idempotency.BeginCompleted,
			Result: data,
		}, nil
	case beginConflictCode:
		return idempotency.BeginResult{}, idempotency.ErrKeyConflict
	default:
		return idempotency.BeginResult{}, fmt.Errorf(
			"idempotency/redis: begin %q returned invalid status %d",
			key,
			code,
		)
	}
}

// Complete atomically replaces an owned active claim with a completed record.
func (s *Store) Complete(
	ctx context.Context,
	request idempotency.CompleteRequest,
) error {
	if err := validateContext(ctx); err != nil {
		return err
	}

	key := s.prefix + request.Key
	completed, err := completeScript.Run(
		ctx,
		s.client,
		[]string{key},
		request.Token,
		request.Result,
		milliseconds(request.TTL),
	).Int64()
	if err != nil {
		return fmt.Errorf("idempotency/redis: complete %q: %w", key, err)
	}
	if completed == 0 {
		return idempotency.ErrClaimLost
	}
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

	key := s.prefix + request.Key
	aborted, err := abortScript.Run(
		ctx,
		s.client,
		[]string{key},
		request.Token,
	).Int64()
	if err != nil {
		return fmt.Errorf("idempotency/redis: abort %q: %w", key, err)
	}
	if aborted == 0 {
		return idempotency.ErrClaimLost
	}
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return idempotency.ErrInvalidContext
	}
	return context.Cause(ctx)
}

func milliseconds(ttl time.Duration) int64 {
	value := ttl.Milliseconds()
	if value < 1 {
		return 1
	}
	return value
}

func resultBytes(value any) ([]byte, error) {
	switch value := value.(type) {
	case string:
		return []byte(value), nil
	case []byte:
		return append([]byte(nil), value...), nil
	default:
		return nil, fmt.Errorf("invalid result type %T", value)
	}
}
