package multi_test

import (
	"testing"

	"github.com/go-fries/fries/gorm/logger/multi/v3"
	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert.NotEmpty(t, multi.Version())
}
