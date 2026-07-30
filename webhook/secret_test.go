package webhook

import (
	"encoding/base64"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecret(t *testing.T) {
	t.Parallel()

	for _, length := range []int{24, 32, 64} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			t.Parallel()

			key := make([]byte, length)
			secret, err := NewSecret(key)

			require.NoError(t, err)
			assert.Len(t, secret.key, length)
		})
	}
}

func TestNewSecretRejectsInvalidLength(t *testing.T) {
	t.Parallel()

	for _, length := range []int{0, 23, 65} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			t.Parallel()

			_, err := NewSecret(make([]byte, length))

			assert.ErrorIs(t, err, ErrInvalidSecret)
		})
	}
}

func TestNewSecretCopiesKey(t *testing.T) {
	t.Parallel()

	key := []byte("01234567890123456789012345678901")
	secret, err := NewSecret(key)
	require.NoError(t, err)

	key[0] = 'x'

	assert.Equal(t, byte('0'), secret.key[0])
}

func TestParseSecret(t *testing.T) {
	t.Parallel()

	key := []byte("01234567890123456789012345678901")
	value := secretPrefix + base64.StdEncoding.EncodeToString(key)

	secret, err := ParseSecret(value)

	require.NoError(t, err)
	assert.Equal(t, key, secret.key)
}

func TestParseSecretRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty":          "",
		"missing prefix": base64.StdEncoding.EncodeToString(make([]byte, 32)),
		"empty key":      secretPrefix,
		"invalid base64": secretPrefix + "***",
		"short key": secretPrefix +
			base64.StdEncoding.EncodeToString(make([]byte, 23)),
	}

	for name, value := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseSecret(value)

			assert.ErrorIs(t, err, ErrInvalidSecret)
		})
	}
}

func FuzzParseSecret(f *testing.F) {
	f.Add("whsec_AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8=")
	f.Add("")

	f.Fuzz(func(t *testing.T, value string) {
		secret, err := ParseSecret(value)
		if err == nil {
			assert.GreaterOrEqual(t, len(secret.key), minSecretLength)
			assert.LessOrEqual(t, len(secret.key), maxSecretLength)
		}
	})
}
