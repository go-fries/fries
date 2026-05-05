package debug

import (
	"io"

	"github.com/davecgh/go-spew/spew"
)

// Dump dumps the given values.
// It is a wrapper for spew.Dump(https://github.com/davecgh/go-spew)
// This is useful for debugging.
func Dump(v ...any) {
	spew.Dump(v...)
}

// Sdump the given values and returns the result.
// It's a wrapper for spew.Sdump(https://github.com/davecgh/go-spew)
// This is useful for debugging.
func Sdump(v ...any) string {
	return spew.Sdump(v...)
}

// Fdump the given values to the given writer.
// It's a wrapper for spew.Fdump(https://github.com/davecgh/go-spew)
// This is useful for debugging.
func Fdump(w io.Writer, v ...any) {
	spew.Fdump(w, v...)
}
