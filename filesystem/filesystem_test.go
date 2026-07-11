package filesystem

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryKind(t *testing.T) {
	assert.False(t, Entry{Kind: EntryKindUnknown}.IsFile())
	assert.False(t, Entry{Kind: EntryKindUnknown}.IsDir())
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

func TestPutOptionsResolveContentLength(t *testing.T) {
	t.Run("explicit", func(t *testing.T) {
		length := int64(7)
		actual, err := (PutOptions{ContentLength: &length}).ResolveContentLength(
			struct{ io.Reader }{Reader: strings.NewReader("content")},
		)
		require.NoError(t, err)
		assert.Equal(t, length, actual)
	})

	t.Run("invalid explicit", func(t *testing.T) {
		length := int64(-1)
		_, err := (PutOptions{ContentLength: &length}).ResolveContentLength(nil)
		assert.ErrorIs(t, err, ErrInvalidContentLength)
	})

	t.Run("explicit mismatch", func(t *testing.T) {
		length := int64(3)
		_, err := (PutOptions{ContentLength: &length}).ResolveContentLength(
			strings.NewReader("content"),
		)
		assert.ErrorIs(t, err, ErrInvalidContentLength)
	})

	t.Run("len", func(t *testing.T) {
		source := strings.NewReader("content")
		_, err := source.Read(make([]byte, 3))
		require.NoError(t, err)

		length, err := (PutOptions{}).ResolveContentLength(source)
		require.NoError(t, err)
		assert.Equal(t, int64(4), length)
	})

	t.Run("seeker", func(t *testing.T) {
		source := &seekOnly{reader: bytes.NewReader([]byte("content"))}
		_, err := source.Seek(3, io.SeekStart)
		require.NoError(t, err)

		length, err := (PutOptions{}).ResolveContentLength(source)
		require.NoError(t, err)
		assert.Equal(t, int64(4), length)

		value := make([]byte, 1)
		_, err = source.Read(value)
		require.NoError(t, err)
		assert.Equal(t, "t", string(value))
	})

	t.Run("unknown", func(t *testing.T) {
		_, err := (PutOptions{}).ResolveContentLength(
			struct{ io.Reader }{Reader: strings.NewReader("content")},
		)
		assert.ErrorIs(t, err, ErrContentLengthRequired)
	})
}

type seekOnly struct {
	reader *bytes.Reader
}

func (s *seekOnly) Read(buffer []byte) (int, error) {
	return s.reader.Read(buffer)
}

func (s *seekOnly) Seek(offset int64, whence int) (int64, error) {
	return s.reader.Seek(offset, whence)
}
