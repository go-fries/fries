package hashing

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// Digest represents the result of a hash operation.
//
// Digest values are immutable. Bytes returns a copy of the underlying bytes,
// so callers cannot mutate the digest through a returned slice. The zero value
// represents an empty digest.
type Digest struct {
	sum []byte
}

// NewDigest creates a Digest by copying sum.
func NewDigest(sum []byte) Digest {
	return Digest{sum: append([]byte(nil), sum...)}
}

// ParseHex decodes value as a hexadecimal digest. It accepts uppercase and
// lowercase hexadecimal characters and returns an error if value is malformed.
func ParseHex(value string) (Digest, error) {
	sum, err := hex.DecodeString(value)
	if err != nil {
		return Digest{}, err
	}

	return Digest{sum: sum}, nil
}

// ParseBase64 decodes value as a standard, padded Base64 digest. It returns an
// error if value is malformed.
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

// Equal reports whether d and other contain the same bytes. For equal-length
// digests, the comparison time does not depend on their contents. Digests with
// different lengths are not equal.
func (d Digest) Equal(other Digest) bool {
	return subtle.ConstantTimeCompare(d.sum, other.sum) == 1
}
