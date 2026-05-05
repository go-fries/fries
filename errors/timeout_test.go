package errors

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeoutError(t *testing.T) {
	err := NewTimeoutError(5*time.Second, nil)
	assert.Equal(t, "timeout after 5s: <nil>", err.Error())
	assert.Nil(t, err.Unwrap())
	assert.True(t, IsTimeoutError(err))
	assert.Equal(t, 5*time.Second, err.Timeout)
	assert.Implements(t, (*error)(nil), err)
}
