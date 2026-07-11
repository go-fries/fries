package filesystem

import (
	"io/fs"
	"strings"
)

// ValidPath reports whether path is a valid logical filesystem path.
// Paths are unrooted, slash-separated, and may use "." for the logical root.
func ValidPath(path string) bool {
	return fs.ValidPath(path) && !strings.Contains(path, `\`)
}

// ValidatePath returns a path error when path is not a valid logical path.
func ValidatePath(path string) error {
	if ValidPath(path) {
		return nil
	}

	return &fs.PathError{Op: "validate", Path: path, Err: ErrInvalidPath}
}

// PathPrefixer maps logical paths to object-storage keys below a root prefix.
type PathPrefixer struct {
	prefix string
}

// NewPathPrefixer creates a logical object-key prefixer.
func NewPathPrefixer(prefix string) *PathPrefixer {
	return &PathPrefixer{prefix: strings.Trim(prefix, "/")}
}

// Prefix maps a logical path to its backend key.
func (p *PathPrefixer) Prefix(path string) string {
	if path == "." {
		return p.prefix
	}
	if p.prefix == "" {
		return path
	}
	return p.prefix + "/" + path
}

// Strip maps a backend key below the configured prefix to a logical path.
func (p *PathPrefixer) Strip(key string) (string, bool) {
	key = strings.TrimPrefix(key, "/")
	if p.prefix == "" {
		if key == "" {
			return ".", true
		}
		return key, true
	}
	if key == p.prefix {
		return ".", true
	}

	path, ok := strings.CutPrefix(key, p.prefix+"/")
	return path, ok
}
