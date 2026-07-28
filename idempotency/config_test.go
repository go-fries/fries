package idempotency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigOptions(t *testing.T) {
	c := newConfig(
		nil,
		WithDefaultExecutionTTL(time.Minute),
		WithDefaultResultTTL(time.Hour),
		WithFinalizationTimeout(10*time.Second),
	)

	assert.Equal(t, time.Minute, c.executionTTL)
	assert.Equal(t, time.Hour, c.resultTTL)
	assert.Equal(t, 10*time.Second, c.finalizationTimeout)
}

func TestInvalidConfigOptionsKeepDefaults(t *testing.T) {
	c := newConfig(
		WithDefaultExecutionTTL(0),
		WithDefaultResultTTL(-1),
		WithFinalizationTimeout(0),
	)

	assert.Equal(t, defaultExecutionTTL, c.executionTTL)
	assert.Equal(t, defaultResultTTL, c.resultTTL)
	assert.Equal(t, defaultFinalizationTimeout, c.finalizationTimeout)
}

func TestExecuteOptions(t *testing.T) {
	c := newConfig(
		WithDefaultExecutionTTL(time.Minute),
		WithDefaultResultTTL(time.Hour),
	)
	execution := newExecuteConfig(
		c,
		nil,
		WithExecutionTTL(2*time.Minute),
		WithResultTTL(2*time.Hour),
		WithFingerprint("request"),
	)

	assert.Equal(t, 2*time.Minute, execution.executionTTL)
	assert.Equal(t, 2*time.Hour, execution.resultTTL)
	assert.Equal(t, "request", execution.fingerprint)
}

func TestInvalidExecuteOptionsKeepDefaults(t *testing.T) {
	c := newConfig()
	execution := newExecuteConfig(
		c,
		WithExecutionTTL(0),
		WithResultTTL(-1),
	)

	assert.Equal(t, defaultExecutionTTL, execution.executionTTL)
	assert.Equal(t, defaultResultTTL, execution.resultTTL)
}
