package redis

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMalformedMessage(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("receive failed: %w", malformedMessage("1-0", assert.AnError))
	assert.True(t, isMalformedMessage(err))
	assert.Equal(t, "1-0", malformedMessageID(err))
	assert.False(t, isMalformedMessage(assert.AnError))
	assert.Empty(t, malformedMessageID(assert.AnError))
	assert.False(t, isMalformedMessage(nil))
	assert.Empty(t, malformedMessageID(nil))
}
