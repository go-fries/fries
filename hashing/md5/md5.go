// Package md5 computes MD5 digests for checksums and legacy compatibility.
//
// MD5 is cryptographically broken. Do not use this package for passwords,
// signatures, certificates, or other security-sensitive purposes.
package md5

import (
	standardmd5 "crypto/md5"
	"io"

	"github.com/go-fries/fries/hashing/v4"
)

var defaultHasher = hashing.New(standardmd5.New)

// New creates an MD5 Hasher.
func New() *hashing.Hasher {
	return hashing.New(standardmd5.New)
}

// Sum computes the MD5 digest of value.
func Sum(value []byte) hashing.Digest {
	return defaultHasher.Sum(value)
}

// SumString computes the MD5 digest of value.
func SumString(value string) hashing.Digest {
	return defaultHasher.SumString(value)
}

// SumReader reads reader until EOF and computes its MD5 digest.
func SumReader(reader io.Reader) (hashing.Digest, error) {
	return defaultHasher.SumReader(reader)
}

// SumFile reads the file at path and computes its MD5 digest.
func SumFile(path string) (hashing.Digest, error) {
	return defaultHasher.SumFile(path)
}
