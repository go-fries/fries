package md5_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-fries/fries/hashing/md5/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMD5Helpers(t *testing.T) {
	const value = "123456"
	const want = "e10adc3949ba59abbe56e057f20f883e"

	assert.Equal(t, want, md5.Sum([]byte(value)).Hex())
	assert.Equal(t, want, md5.SumString(value).Hex())
	assert.Equal(t, want, md5.New().SumString(value).Hex())

	readerDigest, err := md5.SumReader(strings.NewReader(value))
	require.NoError(t, err)
	assert.Equal(t, want, readerDigest.Hex())

	path := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(path, []byte(value), 0o600))
	fileDigest, err := md5.SumFile(path)
	require.NoError(t, err)
	assert.Equal(t, want, fileDigest.Hex())
}
