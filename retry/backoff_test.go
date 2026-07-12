package retry

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		backoff  Backoff
		attempts []int
		want     []time.Duration
	}{
		{
			name:     "none",
			backoff:  NoBackoff(),
			attempts: []int{0, 1, 2},
			want:     []time.Duration{0, 0, 0},
		},
		{
			name:     "fixed",
			backoff:  Fixed(time.Second),
			attempts: []int{0, 1, 2},
			want:     []time.Duration{time.Second, time.Second, time.Second},
		},
		{
			name:     "linear",
			backoff:  Linear(time.Second),
			attempts: []int{0, 1, 2, 3},
			want:     []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second},
		},
		{
			name:     "exponential",
			backoff:  Exponential(time.Second, 0),
			attempts: []int{0, 1, 2, 3},
			want:     []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second},
		},
		{
			name:     "exponential capped",
			backoff:  Exponential(time.Second, 2500*time.Millisecond),
			attempts: []int{1, 2, 3, 4},
			want:     []time.Duration{time.Second, 2 * time.Second, 2500 * time.Millisecond, 2500 * time.Millisecond},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for i, attempt := range tt.attempts {
				assert.Equal(t, tt.want[i], tt.backoff(attempt))
			}
		})
	}
}

func TestBackoffOverflowIsClamped(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Duration(math.MaxInt64), Linear(time.Duration(math.MaxInt64))(2))
	assert.Equal(t, time.Duration(math.MaxInt64), Exponential(time.Duration(math.MaxInt64), 0)(2))
	assert.Equal(t, 5*time.Second, Exponential(4*time.Second, 5*time.Second)(2))
}

func TestJitter(t *testing.T) {
	t.Parallel()

	const (
		base    = time.Second
		maximum = 100 * time.Millisecond
	)
	backoff := Jitter(Fixed(base), maximum)

	for range 100 {
		delay := backoff(1)
		assert.GreaterOrEqual(t, delay, base)
		assert.LessOrEqual(t, delay, base+maximum)
	}

	assert.Zero(t, Jitter(NoBackoff(), maximum)(1))
	assert.Equal(t, base, Jitter(Fixed(base), 0)(1))
	assert.Equal(
		t,
		time.Duration(math.MaxInt64),
		Jitter(Fixed(time.Duration(math.MaxInt64)), time.Second)(1),
	)
}

func TestJitterIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	backoff := Jitter(Fixed(time.Second), time.Millisecond)
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			delay := backoff(1)
			assert.GreaterOrEqual(t, delay, time.Second)
			assert.LessOrEqual(t, delay, time.Second+time.Millisecond)
		})
	}
	wg.Wait()
}

func TestJitterNormalizesNilBackoff(t *testing.T) {
	t.Parallel()

	assert.Zero(t, Jitter(nil, 0)(1))
}

func TestBackoffPanicsOnInvalidDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{name: "negative fixed delay", fn: func() { Fixed(-1) }},
		{name: "negative linear delay", fn: func() { Linear(-1) }},
		{name: "negative exponential initial", fn: func() { Exponential(-1, 0) }},
		{name: "negative exponential maximum", fn: func() { Exponential(0, -1) }},
		{name: "maximum below initial", fn: func() { Exponential(time.Second, time.Millisecond) }},
		{name: "negative jitter maximum", fn: func() { Jitter(NoBackoff(), -1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Panics(t, tt.fn)
		})
	}
}
