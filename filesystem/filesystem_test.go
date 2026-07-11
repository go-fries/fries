package filesystem

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEntryKind(t *testing.T) {
	assert.True(t, Entry{Kind: EntryKindFile}.IsFile())
	assert.False(t, Entry{Kind: EntryKindFile}.IsDir())
	assert.True(t, Entry{Kind: EntryKindDirectory}.IsDir())
	assert.False(t, Entry{Kind: EntryKindDirectory}.IsFile())
}
