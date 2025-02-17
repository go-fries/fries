package fries

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoModVersion(t *testing.T) {
	bytes, err := os.ReadFile("go.mod")
	require.NoError(t, err)
	content := string(bytes)

	assert.NotContains(t, content, "toolchain")

	contents := strings.Split(content, "\n")
	assert.Contains(t, contents, "go 1.23.0")
}
