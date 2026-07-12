// Package md5 computes MD5 digests for checksums and legacy compatibility.
//
// MD5 is cryptographically broken and must not be used for passwords,
// signatures, certificates, or other security-sensitive purposes.
package md5

import (
	standardmd5 "crypto/md5"
	"io"

	"github.com/go-fries/fries/hashing/v4"
)

var defaultHasher = hashing.New(standardmd5.New)

// New returns a reusable [hashing.Hasher] configured to compute MD5 digests.
func New() *hashing.Hasher {
	return hashing.New(standardmd5.New)
}

// Sum returns the MD5 digest of value.
func Sum(value []byte) hashing.Digest {
	return defaultHasher.Sum(value)
}

// SumString returns the MD5 digest of value.
func SumString(value string) hashing.Digest {
	return defaultHasher.SumString(value)
}

// SumReader reads reader until EOF and returns its MD5 digest. It returns
// [hashing.ErrNilReader] for a nil reader and propagates errors reported while
// reading.
func SumReader(reader io.Reader) (hashing.Digest, error) {
	return defaultHasher.SumReader(reader)
}

// SumFile opens the file at path, reads it until EOF, and returns its MD5
// digest. It returns errors encountered while opening, reading, or closing the
// file.
func SumFile(path string) (hashing.Digest, error) {
	return defaultHasher.SumFile(path)
}
