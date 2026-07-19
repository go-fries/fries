package health_test

import (
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
