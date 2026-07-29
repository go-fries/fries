package ratelimit

import (
	"fmt"
	"math"
	"time"
)

const microsecond = int64(time.Microsecond)

// Limit describes the capacity and replenishment rate applied by a [Limiter].
//
// Rate units are replenished during each Period, up to Burst immediately
// available units. All fields must be positive.
type Limit struct {
	// Rate is the number of units replenished during Period.
	Rate int
	// Period is the duration over which Rate units are replenished.
	Period time.Duration
	// Burst is the maximum number of units that can be consumed immediately.
	Burst int
}

// PerSecond returns a limit of rate units per second with a burst equal to
// rate.
func PerSecond(rate int) Limit {
	return Limit{Rate: rate, Period: time.Second, Burst: rate}
}

// PerMinute returns a limit of rate units per minute with a burst equal to
// rate.
func PerMinute(rate int) Limit {
	return Limit{Rate: rate, Period: time.Minute, Burst: rate}
}

// PerHour returns a limit of rate units per hour with a burst equal to rate.
func PerHour(rate int) Limit {
	return Limit{Rate: rate, Period: time.Hour, Burst: rate}
}

func validateLimit(limit Limit) error {
	if limit.Rate <= 0 {
		return fmt.Errorf("%w: rate must be greater than zero", ErrInvalidLimit)
	}
	if limit.Period <= 0 {
		return fmt.Errorf("%w: period must be greater than zero", ErrInvalidLimit)
	}
	if limit.Burst <= 0 {
		return fmt.Errorf("%w: burst must be greater than zero", ErrInvalidLimit)
	}

	rate := int64(limit.Rate)
	periodNanos := int64(limit.Period)
	if rate > periodNanos/int64(time.Microsecond) {
		return fmt.Errorf(
			"%w: replenishment interval must be at least one microsecond",
			ErrInvalidLimit,
		)
	}

	intervalNanos := divideRoundUp(periodNanos, rate)
	interval := divideRoundUp(intervalNanos, int64(time.Microsecond))
	if int64(limit.Burst) > math.MaxInt64/interval ||
		interval*int64(limit.Burst) > math.MaxInt64/microsecond {
		return fmt.Errorf("%w: burst duration overflows time.Duration", ErrInvalidLimit)
	}
	return nil
}

func divideRoundUp(value, divisor int64) int64 {
	result := value / divisor
	if value%divisor != 0 {
		result++
	}
	return result
}
