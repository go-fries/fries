package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIs(t *testing.T) {
	assert.True(t, Is("abc", "abc"))
	assert.False(t, Is("abcc", "abc"))
	assert.True(t, Is("ab*", "abc"))
	assert.True(t, Is("ab*", "ab"))
	assert.True(t, Is("ab/*", "ab/cc"))
	assert.True(t, Is("ab/*", "ab/"))
	assert.True(t, Is("*", "ab"))
	assert.True(t, Is("*", ""))
	assert.True(t, Is("ab/*", "ab/cc/dd"))
	assert.True(t, Is("*dd/", "ab/cc/dd/"))
	assert.False(t, Is("*dd/d", "dd/"))
}

func TestInSlice(t *testing.T) {
	assert.True(t, InSlice([]string{"1", "2"}, "1"))
	assert.True(t, InSlice([]string{"1", "2"}, "2"))
	assert.False(t, InSlice([]string{"1", "2"}, "3"))
	assert.False(t, InSlice([]string{"1", "2"}, "12"))
}

func TestMD5(t *testing.T) {
	assert.Equal(t, "900150983cd24fb0d6963f7d28e17f72", MD5("abc"))
}

func TestSHA1(t *testing.T) {
	assert.Equal(t, "a9993e364706816aba3e25717850c26c9cd0d89d", SHA1("abc"))
}

func TestReverse(t *testing.T) {
	assert.Equal(t, "cba", Reverse("abc"))
}

func TestReplace(t *testing.T) {
	assert.Equal(t, "bbbbcc", Replace("aabbcc", "a", "b"))
}

func TestStrShuffle(t *testing.T) {
	assert.True(t, InSlice([]string{"abc", "acb", "bac", "bca", "cab", "cba"}, Shuffle("abc")))
}

func TestRandom(t *testing.T) {
	r1, r2 := Random(10), Random(10)

	assert.Equal(t, 10, len(r1))
	assert.Equal(t, 10, len(r2))
	assert.NotEqual(t, r1, r2)
}

func TestLength(t *testing.T) {
	assert.Equal(t, 3, Len("abc"))
	assert.Equal(t, 5, Len("^*&%*"))
	assert.Equal(t, 4, Len("张三李四"))

	assert.Equal(t, 12, len("张三李四"))
}

func TestIsUUID(t *testing.T) {
	assert.True(t, IsUUID("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"))
	assert.False(t, IsUUID("f81d4fae-7dec-11d0-a765-00a0c91e6bf"))
	assert.False(t, IsUUID("f81d4fae-7dec-11d0-a765-00a0c91e6bf6a"))
}

func TestUUID(t *testing.T) {
	uuid := UUID()

	assert.NotEmpty(t, uuid)
	assert.Equal(t, 36, len(uuid))
	assert.True(t, IsUUID(uuid))
}

func TestAfter(t *testing.T) {
	tests := []struct {
		name, s, sep, want string
	}{
		{"", "Hello, World!", ",", " World!"},
		{"", "Hello, World!", "", "Hello, World!"},
		{"", "", "Hello", ""},
		{"", "张三李四", "三", "李四"},
		{"", "张三张三李四", "三", "张三李四"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, After(tt.s, tt.sep))
	}
}

func TestBefore(t *testing.T) {
	tests := []struct {
		name, s, sep, want string
	}{
		{"", "Hello, World!", ",", "Hello"},
		{"", "Hello, World!", "", "Hello, World!"},
		{"", "", "Hello", ""},
		{"", "张三李四", "三", "张"},
		{"", "张三张三李四", "三", "张"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, Before(tt.s, tt.sep))
	}
}

func TestSubstrCount(t *testing.T) {
	assert.Equal(t, 5, SubstrCount("babababbaaba", "a", 0, 10))
	assert.Equal(t, 0, SubstrCount("babababbaaba", "a", -1, 10))
	assert.Equal(t, 0, SubstrCount("babababbaaba", "a", 15, 10))
	assert.Equal(t, 6, SubstrCount("babababbaaba", "a", 0, Len("babababbaaba")))
	assert.Equal(t, 2, SubstrCount("121212312", "1", 1, 5))
}
