package ratelimit

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimitHelpers(t *testing.T) {
	assert.Equal(t, Limit{Rate: 2, Period: time.Second, Burst: 2}, PerSecond(2))
	assert.Equal(t, Limit{Rate: 3, Period: time.Minute, Burst: 3}, PerMinute(3))
	assert.Equal(t, Limit{Rate: 4, Period: time.Hour, Burst: 4}, PerHour(4))
}

func TestValidateLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit Limit
		valid bool
	}{
		{
			name:  "valid",
			limit: Limit{Rate: 3, Period: 10 * time.Microsecond, Burst: 2},
			valid: true,
		},
		{
			name:  "zero rate",
			limit: Limit{Period: time.Second, Burst: 1},
		},
		{
			name:  "negative rate",
			limit: Limit{Rate: -1, Period: time.Second, Burst: 1},
		},
		{
			name:  "zero period",
			limit: Limit{Rate: 1, Burst: 1},
		},
		{
			name:  "negative period",
			limit: Limit{Rate: 1, Period: -time.Second, Burst: 1},
		},
		{
			name:  "zero burst",
			limit: Limit{Rate: 1, Period: time.Second},
		},
		{
			name:  "negative burst",
			limit: Limit{Rate: 1, Period: time.Second, Burst: -1},
		},
		{
			name:  "sub-microsecond interval",
			limit: Limit{Rate: 2, Period: time.Microsecond, Burst: 1},
		},
		{
			name: "burst duration overflow",
			limit: Limit{
				Rate:   1,
				Period: time.Duration(math.MaxInt64),
				Burst:  2,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLimit(test.limit)
			if test.valid {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrInvalidLimit)
		})
	}
}
