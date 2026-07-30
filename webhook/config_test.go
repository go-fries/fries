package webhook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSignerConfiguration(t *testing.T) {
	t.Parallel()

	primary := mustSecret(t, 0)
	additional := mustSecret(t, 1)

	signer, err := NewSigner(
		primary,
		nil,
		WithAdditionalSigningSecrets(additional),
	)

	require.NoError(t, err)
	require.Len(t, signer.secrets, 2)
	assert.Equal(t, primary.key, signer.secrets[0])
	assert.Equal(t, additional.key, signer.secrets[1])

	primary.key[0] = 9
	additional.key[0] = 9
	assert.Equal(t, byte(0), signer.secrets[0][0])
	assert.Equal(t, byte(1), signer.secrets[1][0])
}

func TestNewSignerRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	_, err := NewSigner(Secret{})
	assert.ErrorIs(t, err, ErrInvalidSecret)

	_, err = NewSigner(
		mustSecret(t, 0),
		WithAdditionalSigningSecrets(Secret{}),
	)
	assert.ErrorIs(t, err, ErrInvalidSecret)
}

func TestNewVerifierConfiguration(t *testing.T) {
	t.Parallel()

	primary := mustSecret(t, 0)
	additional := mustSecret(t, 1)

	verifier, err := NewVerifier(
		primary,
		nil,
		WithTolerance(time.Minute),
		WithAdditionalVerificationSecrets(additional),
	)

	require.NoError(t, err)
	assert.Equal(t, time.Minute, verifier.tolerance)
	require.Len(t, verifier.secrets, 2)
	assert.Equal(t, primary.key, verifier.secrets[0])
	assert.Equal(t, additional.key, verifier.secrets[1])

	primary.key[0] = 9
	additional.key[0] = 9
	assert.Equal(t, byte(0), verifier.secrets[0][0])
	assert.Equal(t, byte(1), verifier.secrets[1][0])
}

func TestNewVerifierRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	_, err := NewVerifier(Secret{})
	assert.ErrorIs(t, err, ErrInvalidSecret)

	_, err = NewVerifier(
		mustSecret(t, 0),
		WithAdditionalVerificationSecrets(Secret{}),
	)
	assert.ErrorIs(t, err, ErrInvalidSecret)

	for _, tolerance := range []time.Duration{0, -time.Second} {
		_, err = NewVerifier(
			mustSecret(t, 0),
			WithTolerance(tolerance),
		)
		assert.ErrorIs(t, err, ErrInvalidTolerance)
	}
}

func mustSecret(t testing.TB, value byte) Secret {
	t.Helper()

	secret, err := NewSecret(bytesOf(value, 32))
	require.NoError(t, err)
	return secret
}

func bytesOf(value byte, length int) []byte {
	result := make([]byte, length)
	for index := range result {
		result[index] = value
	}
	return result
}
