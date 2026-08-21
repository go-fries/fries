package ptr_test

import (
	"testing"
	"time"

	"github.com/go-fries/fries/ptr/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPtr(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		got := ptr.Ptr("foo")
		require.NotNil(t, got)
		assert.Equal(t, "foo", *got)
	})

	t.Run("integer", func(t *testing.T) {
		got := ptr.Ptr(10)
		require.NotNil(t, got)
		assert.Equal(t, 10, *got)
	})

	t.Run("struct", func(t *testing.T) {
		type value struct {
			Name string
		}

		got := ptr.Ptr(value{Name: "foo"})
		require.NotNil(t, got)
		assert.Equal(t, "foo", got.Name)
	})

	t.Run("time", func(t *testing.T) {
		now := time.Now()
		got := ptr.Ptr(now)
		require.NotNil(t, got)
		assert.Equal(t, now, *got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		got := ptr.Ptr[*int](nil)
		require.NotNil(t, got)
		assert.Nil(t, *got)
	})

	t.Run("independent values", func(t *testing.T) {
		first := ptr.Ptr(1)
		second := ptr.Ptr(1)
		assert.NotSame(t, first, second)
	})
}

func TestValue(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		got, ok := ptr.Value(new("foo"))
		assert.True(t, ok)
		assert.Equal(t, "foo", got)
	})

	t.Run("zero value", func(t *testing.T) {
		got, ok := ptr.Value(new(""))
		assert.True(t, ok)
		assert.Empty(t, got)
	})

	t.Run("nil pointer", func(t *testing.T) {
		got, ok := ptr.Value[string](nil)
		assert.False(t, ok)
		assert.Empty(t, got)
	})

	t.Run("pointer containing nil", func(t *testing.T) {
		got, ok := ptr.Value(new((*int)(nil)))
		assert.True(t, ok)
		assert.Nil(t, got)
	})
}

func TestOr(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		assert.Equal(t, "foo", ptr.Or(new("foo"), "fallback"))
	})

	t.Run("zero value", func(t *testing.T) {
		assert.Empty(t, ptr.Or(new(""), "fallback"))
	})

	t.Run("nil pointer", func(t *testing.T) {
		assert.Equal(t, "fallback", ptr.Or(nil, "fallback"))
	})

	t.Run("pointer containing nil", func(t *testing.T) {
		fallback := new(10)
		assert.Nil(t, ptr.Or(new((*int)(nil)), fallback))
	})
}
