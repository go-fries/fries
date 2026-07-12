package hashing_test

import (
	"regexp"
	"testing"

	"github.com/go-fries/fries/hashing/v4"
	"github.com/stretchr/testify/assert"
)

func TestVersionIsSemver(t *testing.T) {
	pattern := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

	assert.Regexp(t, pattern, hashing.Version())
}
