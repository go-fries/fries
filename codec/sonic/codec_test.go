package sonic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSonic(t *testing.T) {
	c1, c2 := Codec, Codec

	assert.Same(t, c1, c2)

	data := map[string]any{
		"foo": "bar",
	}

	// marshal
	bytes, err := c1.Marshal(data)
	assert.NoError(t, err)

	// unmarshal
	dest := make(map[string]any)
	assert.NoError(t, c1.Unmarshal(bytes, &dest))
}
