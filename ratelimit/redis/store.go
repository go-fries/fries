package redis

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-fries/fries/ratelimit/v4"
	goredis "github.com/redis/go-redis/v9"
)

// Store keeps rate-limit state in Redis.
//
// A Store is safe for concurrent use.
type Store struct {
	client goredis.UniversalClient
	prefix string
}

var _ ratelimit.Store = (*Store)(nil)

// New creates a Redis-backed [Store].
//
// New panics if client is nil.
func New(client goredis.UniversalClient, options ...Option) *Store {
	if client == nil {
		panic("ratelimit/redis: nil client")
	}
	c := newConfig(options...)
	return &Store{
		client: client,
		prefix: c.prefix,
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

	key := s.prefix + request.Key
	raw, err := takeScript.Run(
		ctx,
		s.client,
		[]string{key},
		emissionInterval(request.Limit),
		request.Limit.Burst,
		request.Cost,
	).Slice()
	if err != nil {
		return ratelimit.Decision{}, fmt.Errorf(
			"ratelimit/redis: take %q: %w",
			key,
			err,
		)
	}

	decision, err := parseDecision(raw, request.Limit)
	if err != nil {
		return ratelimit.Decision{}, fmt.Errorf(
			"ratelimit/redis: take %q: %w",
			key,
			err,
		)
	}
	return decision, nil
}

// Reset removes the stored state for key.
func (s *Store) Reset(ctx context.Context, key string) error {
	if err := validateContext(ctx); err != nil {
		return err
	}

	redisKey := s.prefix + key
	if err := s.client.Del(ctx, redisKey).Err(); err != nil {
		return fmt.Errorf("ratelimit/redis: reset %q: %w", redisKey, err)
	}
	return nil
}

func parseDecision(
	raw []any,
	limit ratelimit.Limit,
) (ratelimit.Decision, error) {
	if len(raw) != 4 {
		return ratelimit.Decision{}, fmt.Errorf(
			"invalid script result length %d",
			len(raw),
		)
	}

	values := make([]int64, len(raw))
	for i, value := range raw {
		integer, ok := value.(int64)
		if !ok {
			return ratelimit.Decision{}, fmt.Errorf(
				"invalid script result value %d type %T",
				i,
				value,
			)
		}
		values[i] = integer
	}
	if values[0] != 0 && values[0] != 1 {
		return ratelimit.Decision{}, fmt.Errorf(
			"invalid script allowed value %d",
			values[0],
		)
	}
	maxDurationMicros := int64(math.MaxInt64) / int64(time.Microsecond)
	if values[1] < 0 || values[1] > int64(limit.Burst) ||
		values[2] < 0 || values[2] > maxDurationMicros ||
		values[3] < 0 || values[3] > maxDurationMicros {
		return ratelimit.Decision{}, fmt.Errorf(
			"invalid script result values %v",
			values,
		)
	}

	return ratelimit.Decision{
		Limit:      limit,
		Allowed:    values[0] == 1,
		Remaining:  int(values[1]),
		RetryAfter: time.Duration(values[2]) * time.Microsecond,
		ResetAfter: time.Duration(values[3]) * time.Microsecond,
	}, nil
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

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return ratelimit.ErrInvalidContext
	}
	return context.Cause(ctx)
}
