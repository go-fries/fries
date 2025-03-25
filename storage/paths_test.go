package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathPrefixer_Prefix(t *testing.T) {
	prefixer := NewPathPrefixer("prefix")

	tests := []struct {
		path     string
		expected string
	}{
		{"path", "prefix/path"},
		{"/path", "prefix/path"},
		{"//path", "prefix/path"},
		{"path/", "prefix/path/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.expected, prefixer.Prefix(test.path))
		})
	}
}

func TestPathPrefixer_PrefixEmptyPrefix(t *testing.T) {
	prefixer := NewPathPrefixer("")

	tests := []struct {
		path     string
		expected string
	}{
		{"path", "path"},
		{"/path", "path"},
		{"//path", "path"},
		{"path/", "path/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.expected, prefixer.Prefix(test.path))
		})
	}
}

func TestPathPrefixer_PrefixWithSeparator(t *testing.T) {
	prefixer := NewPathPrefixer("prefix", WithPathPrefixerSeparator("::"))

	tests := []struct {
		path     string
		expected string
	}{
		{"path", "prefix::path"},
		{"/path", "prefix::path"},
		{"//path", "prefix::path"},
		{"path/", "prefix::path/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.expected, prefixer.Prefix(test.path))
		})
	}
}

func TestPathPrefixer_PrefixWithSamePrefixAndSeparator(t *testing.T) {
	prefixer := NewPathPrefixer("", WithPathPrefixerSeparator(""))

	tests := []struct {
		path     string
		expected string
	}{
		{"path", "path"},
		{"/path", "path"},
		{"//path", "path"},
		{"path/", "path/"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			assert.Equal(t, test.expected, prefixer.Prefix(test.path))
		})
	}
}
