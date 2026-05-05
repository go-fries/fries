package md5

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMd5Hasher(t *testing.T) {
	md5 := New()

	value := "123456"
	hashedValue, err := md5.Make(value)

	assert.Nil(t, err)
	assert.True(t, md5.Check(value, hashedValue))

	assert.True(t, md5.Check(value, md5.MustMake(value)))

	md5Two := New()
	assert.Same(t, md5, md5Two)
}
