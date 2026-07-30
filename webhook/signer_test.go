package webhook

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTime = time.Unix(1_700_000_000, 0).UTC()

func TestSignerSign(t *testing.T) {
	t.Parallel()

	key := make([]byte, 32)
	for index := range key {
		key[index] = byte(index)
	}
	secret, err := NewSecret(key)
	require.NoError(t, err)
	signer, err := NewSigner(secret)
	require.NoError(t, err)
	signer.now = func() time.Time { return testTime }

	headers, err := signer.Sign(
		"msg_test",
		[]byte(`{"type":"example.created"}`),
	)

	require.NoError(t, err)
	assert.Equal(t, "msg_test", headers.Get(HeaderID))
	assert.Equal(t, "1700000000", headers.Get(HeaderTimestamp))
	assert.Equal(
		t,
		"v1,+Ic57Le4o6523vOytAQ+1WEe0iMI81lYzLep8kBQTq8=",
		headers.Get(HeaderSignature),
	)
}

func TestSignerSignWithAdditionalSecrets(t *testing.T) {
	t.Parallel()

	signer, err := NewSigner(
		mustSecret(t, 0),
		WithAdditionalSigningSecrets(mustSecret(t, 1)),
	)
	require.NoError(t, err)
	signer.now = func() time.Time { return testTime }

	headers, err := signer.Sign("msg_rotation", []byte("payload"))

	require.NoError(t, err)
	signatures := strings.Fields(headers.Get(HeaderSignature))
	require.Len(t, signatures, 2)
	assert.NotEqual(t, signatures[0], signatures[1])
	assert.True(t, strings.HasPrefix(signatures[0], "v1,"))
	assert.True(t, strings.HasPrefix(signatures[1], "v1,"))
}

func TestSignerSignRejectsInvalidMessageID(t *testing.T) {
	t.Parallel()

	signer, err := NewSigner(mustSecret(t, 0))
	require.NoError(t, err)

	for _, messageID := range []string{
		"",
		"msg.with.dot",
		"msg with space",
		"msg\nheader",
		"消息",
	} {
		_, err = signer.Sign(messageID, nil)
		assert.ErrorIs(t, err, ErrInvalidMessageID)
	}
}

func TestSignerConcurrentUse(t *testing.T) {
	t.Parallel()

	signer, err := NewSigner(mustSecret(t, 0))
	require.NoError(t, err)
	signer.now = func() time.Time { return testTime }

	var wait sync.WaitGroup
	for range 50 {
		wait.Go(func() {
			headers, signErr := signer.Sign("msg_concurrent", []byte("payload"))
			assert.NoError(t, signErr)
			assert.NotEmpty(t, headers.Get(HeaderSignature))
		})
	}
	wait.Wait()
}
