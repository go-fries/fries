package ctxvaluer_test

import (
	"testing"

	"github.com/go-fries/fries/log/slog/ctxvaluer/v4"
	"github.com/stretchr/testify/assert"
)

func TestVersionSemver(t *testing.T) {
	assert.Regexp(t, `^\d+\.\d+\.\d+$`, ctxvaluer.Version())
}
