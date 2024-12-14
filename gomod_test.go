package components

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

	contents := strings.Split(string(bytes), "\n")
	assert.Subset(t, contents, []string{"go 1.22.0"})
}
