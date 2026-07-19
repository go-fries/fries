package health

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	c := newConfig(nil, WithTimeout(2*time.Second), WithConcurrency(8))

	assert.Equal(t, 2*time.Second, c.timeout)
	assert.Equal(t, 8, c.concurrency)
}

func TestNewConfigIgnoresInvalidValues(t *testing.T) {
	c := newConfig(WithTimeout(0), WithTimeout(-time.Second), WithConcurrency(0), WithConcurrency(-1))

	assert.Equal(t, defaultTimeout, c.timeout)
	assert.Equal(t, defaultConcurrency, c.concurrency)
}
