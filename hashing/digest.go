package hashing

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// Digest is the result of a hash operation.
//
// A Digest does not expose its internal byte slice. Bytes returns a copy, so a
// Digest can be safely passed between callers without accidental mutation.
type Digest struct {
	sum []byte
}

// NewDigest creates a Digest by copying sum.
func NewDigest(sum []byte) Digest {
	return Digest{sum: append([]byte(nil), sum...)}
}

// ParseHex decodes a hexadecimal digest.
func ParseHex(value string) (Digest, error) {
	sum, err := hex.DecodeString(value)
	if err != nil {
		return Digest{}, err
	}

	return Digest{sum: sum}, nil
}

// ParseBase64 decodes a standard, padded Base64 digest.
func ParseBase64(value string) (Digest, error) {
	sum, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return Digest{}, err
	}

	return Digest{sum: sum}, nil
}

// Bytes returns a copy of the digest bytes.
func (d Digest) Bytes() []byte {
	return append([]byte(nil), d.sum...)
}

// Hex returns the lowercase hexadecimal encoding of the digest.
func (d Digest) Hex() string {
	return hex.EncodeToString(d.sum)
}

// Base64 returns the standard, padded Base64 encoding of the digest.
func (d Digest) Base64() string {
	return base64.StdEncoding.EncodeToString(d.sum)
}

// Equal reports whether d and other contain the same digest bytes.
// Comparison time depends on the digest lengths but not their contents.
func (d Digest) Equal(other Digest) bool {
	return subtle.ConstantTimeCompare(d.sum, other.sum) == 1
}
