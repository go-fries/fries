package health_test

import (
	"errors"
	"testing"

	"github.com/go-fries/fries/health/v4"
	"github.com/stretchr/testify/assert"
)

func TestResultHealthy(t *testing.T) {
	assert.True(t, (health.Result{}).Healthy())
	assert.False(t, (health.Result{Err: assert.AnError}).Healthy())
}

func TestReportHealthy(t *testing.T) {
	assert.True(t, (health.Report{}).Healthy())
	assert.True(t, (health.Report{
		Results: []health.Result{{Name: "first"}, {Name: "second"}},
	}).Healthy())
	assert.False(t, (health.Report{
		Results: []health.Result{{Name: "first"}, {Name: "second", Err: assert.AnError}},
	}).Healthy())
}

func TestPanicError(t *testing.T) {
	cause := errors.New("panic")
	err := &health.PanicError{Value: cause}

	assert.Equal(t, "health: checker panic: panic", err.Error())
	assert.ErrorIs(t, err, cause)
	assert.NoError(t, (&health.PanicError{Value: "panic"}).Unwrap())
}
