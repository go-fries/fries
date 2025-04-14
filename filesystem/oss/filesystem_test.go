package oss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilesystem(t *testing.T) {
	assert.NotNil(t, New(nil, "bucket"))
}
