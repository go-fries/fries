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

func TestListOptionsNormalize(t *testing.T) {
	assert.Equal(t, DefaultListLimit, (ListOptions{}).Normalize().Limit)
	assert.Equal(t, 10, (ListOptions{Limit: 10}).Normalize().Limit)
	assert.Equal(t, MaxListLimit, (ListOptions{Limit: MaxListLimit + 1}).Normalize().Limit)
}
