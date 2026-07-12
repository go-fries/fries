package hashing_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"hash"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/go-fries/fries/hashing/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasherSupportsCommonInputs(t *testing.T) {
	value := []byte("hello")
	want := sha256.Sum256(value)
	hasher := hashing.New(sha256.New)

	assert.Equal(t, want[:], hasher.Sum(value).Bytes())
	assert.Equal(t, want[:], hasher.SumString(string(value)).Bytes())

	readerDigest, err := hasher.SumReader(bytes.NewReader(value))
	require.NoError(t, err)
	assert.Equal(t, want[:], readerDigest.Bytes())

	path := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(path, value, 0o600))
	fileDigest, err := hasher.SumFile(path)
	require.NoError(t, err)
	assert.Equal(t, want[:], fileDigest.Bytes())
}

func TestHasherCanBeSharedConcurrently(t *testing.T) {
	hasher := hashing.New(sha256.New)
	want := hasher.SumString("shared")
	var waitGroup sync.WaitGroup

	for range 100 {
		waitGroup.Go(func() {
			assert.True(t, want.Equal(hasher.SumString("shared")))
		})
	}

	waitGroup.Wait()
}

func TestHasherReturnsInputErrors(t *testing.T) {
	hasher := hashing.New(sha256.New)

	_, err := hasher.SumReader(nil)
	require.ErrorIs(t, err, hashing.ErrNilReader)

	wantErr := errors.New("read failed")
	_, err = hasher.SumReader(errorReader{err: wantErr})
	require.ErrorIs(t, err, wantErr)

	_, err = hasher.SumFile(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestHasherRejectsInvalidConstructors(t *testing.T) {
	assert.PanicsWithValue(t, "hashing: nil hash constructor", func() {
		hashing.New(nil)
	})

	hasher := hashing.New(func() hash.Hash { return nil })
	assert.PanicsWithValue(t, "hashing: hash constructor returned nil", func() {
		hasher.SumString("value")
	})
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

func TestDigestEncodingsAndParsing(t *testing.T) {
	digest := hashing.NewDigest([]byte{0x01, 0x02, 0xfe, 0xff})

	assert.Equal(t, "0102feff", digest.Hex())
	assert.Equal(t, "AQL+/w==", digest.Base64())

	fromHex, err := hashing.ParseHex(digest.Hex())
	require.NoError(t, err)
	assert.True(t, digest.Equal(fromHex))

	fromBase64, err := hashing.ParseBase64(digest.Base64())
	require.NoError(t, err)
	assert.True(t, digest.Equal(fromBase64))

	_, err = hashing.ParseHex("not hex")
	require.Error(t, err)
	_, err = hashing.ParseBase64("not base64")
	require.Error(t, err)
}

func TestDigestOwnsItsBytes(t *testing.T) {
	sum := []byte{1, 2, 3}
	digest := hashing.NewDigest(sum)
	sum[0] = 9

	got := digest.Bytes()
	got[1] = 9

	assert.Equal(t, []byte{1, 2, 3}, digest.Bytes())
	assert.False(t, digest.Equal(hashing.NewDigest([]byte{1, 2})))
	assert.False(t, digest.Equal(hashing.NewDigest([]byte{1, 2, 4})))
}

func TestSumReaderUsesTheWholeStream(t *testing.T) {
	hasher := hashing.New(sha256.New)
	parts := []string{"large", " streamed", " value"}

	digest, err := hasher.SumReader(strings.NewReader(strings.Join(parts, "")))
	require.NoError(t, err)
	assert.True(t, digest.Equal(hasher.SumString(strings.Join(parts, ""))))
}
