package hashing

import (
	"errors"
	"hash"
	"io"
	"os"
)

// NewHash returns a new [hash.Hash].
//
// Each call must return a distinct, non-nil hash.Hash. A NewHash passed to New
// must be safe to call concurrently.
type NewHash func() hash.Hash

// Hasher computes independent digests using a new [hash.Hash] for each
// operation. A Hasher is safe for concurrent use.
type Hasher struct {
	newHash NewHash
}

// New returns a reusable Hasher backed by newHash.
//
// New panics if newHash is nil. Operations on the returned Hasher panic if
// newHash returns nil. See NewHash for the constructor contract.
func New(newHash NewHash) *Hasher {
	if newHash == nil {
		panic("hashing: nil hash constructor")
	}

	return &Hasher{newHash: newHash}
}

// Sum returns the digest of value.
func (h *Hasher) Sum(value []byte) Digest {
	hashValue := h.create()
	_, _ = hashValue.Write(value)

	return NewDigest(hashValue.Sum(nil))
}

// SumString returns the digest of value.
func (h *Hasher) SumString(value string) Digest {
	return h.Sum([]byte(value))
}

// SumReader reads reader until EOF and returns its digest. It returns
// ErrNilReader for a nil reader and propagates errors reported while reading.
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

// SumFile opens the file at path, reads it until EOF, and returns its digest.
// It returns errors encountered while opening, reading, or closing the file.
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
