package storage

import "strings"

type PathPrefixer struct {
	prefix    string
	separator string
}

type PathPrefixerOption func(*PathPrefixer)

func WithPathPrefixerSeparator(separator string) PathPrefixerOption {
	return func(p *PathPrefixer) {
		p.separator = separator
	}
}

func NewPathPrefixer(prefix string, opts ...PathPrefixerOption) *PathPrefixer {
	prefixer := &PathPrefixer{
		prefix:    strings.TrimRight(prefix, "\\/"),
		separator: "/",
	}
	for _, opt := range opts {
		opt(prefixer)
	}

	if prefixer.prefix != "" || prefix == prefixer.separator {
		prefixer.prefix += prefixer.separator
	}

	return prefixer
}

func (p *PathPrefixer) Prefix(path string) string {
	return p.prefix + strings.TrimLeft(path, "\\/")
}
