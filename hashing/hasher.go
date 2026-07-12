package hashing

import (
	"errors"
	"hash"
	"io"
	"os"
)

// Hasher computes independent digests using a new hash.Hash for every
// operation. A Hasher is safe for concurrent use when its constructor is safe
// for concurrent use.
type Hasher struct {
	newHash func() hash.Hash
}

// New creates a reusable Hasher backed by newHash.
//
// New panics when newHash is nil. The constructor must return a new, non-nil
// hash.Hash each time it is called.
func New(newHash func() hash.Hash) *Hasher {
	if newHash == nil {
		panic("hashing: nil hash constructor")
	}

	return &Hasher{newHash: newHash}
}

// Sum computes the digest of value.
func (h *Hasher) Sum(value []byte) Digest {
	hashValue := h.create()
	_, _ = hashValue.Write(value)

	return NewDigest(hashValue.Sum(nil))
}

// SumString computes the digest of value.
func (h *Hasher) SumString(value string) Digest {
	return h.Sum([]byte(value))
}

// SumReader reads reader until EOF and computes its digest.
func (h *Hasher) SumReader(reader io.Reader) (Digest, error) {
	if reader == nil {
		return Digest{}, ErrNilReader
	}

	hashValue := h.create()
	if _, err := io.Copy(hashValue, reader); err != nil {
		return Digest{}, err
	}

	return NewDigest(hashValue.Sum(nil)), nil
}

// SumFile reads the file at path and computes its digest.
func (h *Hasher) SumFile(path string) (digest Digest, err error) {
	file, err := os.Open(path)
	if err != nil {
		return Digest{}, err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()

	return h.SumReader(file)
}

func (h *Hasher) create() hash.Hash {
	hashValue := h.newHash()
	if hashValue == nil {
		panic("hashing: hash constructor returned nil")
	}

	return hashValue
}
