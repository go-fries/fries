package hashing_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-fries/fries/v3/hashing"
	_ "github.com/go-fries/fries/v3/hashing/md5"
)

func TestHasher_New(t *testing.T) {
	hashedValue, err := hashing.MD5.New().Make("123456")
	assert.NoError(t, err)

	assert.True(t, hashing.MD5.New().Check("123456", hashedValue))

	// unknown
	assert.Panics(t, func() {
		hashing.Hash(999999).New()
	})
}

func TestHasher_Instance(t *testing.T) {
	h1 := hashing.MD5.New()
	h2 := hashing.MD5.New()

	assert.Same(t, h1, h2)
}
