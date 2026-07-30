package webhook

import (
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifierVerify(t *testing.T) {
	t.Parallel()

	signer, verifier := newTestPair(t)
	payload := []byte(`{"type":"example.created"}`)
	headers, err := signer.Sign("msg_test", payload)
	require.NoError(t, err)

	metadata, err := verifier.Verify(headers, payload)

	require.NoError(t, err)
	assert.Equal(t, "msg_test", metadata.ID)
	assert.Equal(t, testTime, metadata.Timestamp)
}

func TestVerifierSupportsMultipleHeadersAndVersions(t *testing.T) {
	t.Parallel()

	signer, verifier := newTestPair(t)
	payload := []byte("payload")
	headers, err := signer.Sign("msg_multiple", payload)
	require.NoError(t, err)
	validSignature := headers.Get(HeaderSignature)
	headers.Set(HeaderSignature, "v2,unknown")
	headers.Add(HeaderSignature, "invalid")
	headers.Add(HeaderSignature, validSignature)

	metadata, err := verifier.Verify(headers, payload)

	require.NoError(t, err)
	assert.Equal(t, "msg_multiple", metadata.ID)
}

func TestVerifierSupportsSecretRotation(t *testing.T) {
	t.Parallel()

	current := mustSecret(t, 0)
	previous := mustSecret(t, 1)
	signer, err := NewSigner(previous)
	require.NoError(t, err)
	signer.now = func() time.Time { return testTime }
	verifier, err := NewVerifier(
		current,
		WithAdditionalVerificationSecrets(previous),
	)
	require.NoError(t, err)
	verifier.now = func() time.Time { return testTime }
	headers, err := signer.Sign("msg_rotation", []byte("payload"))
	require.NoError(t, err)

	_, err = verifier.Verify(headers, []byte("payload"))

	assert.NoError(t, err)
}

func TestVerifierTimestampTolerance(t *testing.T) {
	t.Parallel()

	secret := mustSecret(t, 0)
	signer, err := NewSigner(secret)
	require.NoError(t, err)
	verifier, err := NewVerifier(secret, WithTolerance(time.Minute))
	require.NoError(t, err)
	verifier.now = func() time.Time { return testTime }

	tests := map[string]struct {
		signedAt time.Time
		wantErr  error
	}{
		"past boundary": {
			signedAt: testTime.Add(-time.Minute),
		},
		"future boundary": {
			signedAt: testTime.Add(time.Minute),
		},
		"too old": {
			signedAt: testTime.Add(-time.Minute - time.Second),
			wantErr:  ErrTimestampOutsideTolerance,
		},
		"too far in future": {
			signedAt: testTime.Add(time.Minute + time.Second),
			wantErr:  ErrTimestampOutsideTolerance,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			signer.now = func() time.Time { return test.signedAt }
			headers, signErr := signer.Sign("msg_time", nil)
			require.NoError(t, signErr)

			_, verifyErr := verifier.Verify(headers, nil)

			assert.ErrorIs(t, verifyErr, test.wantErr)
		})
	}
}

func TestVerifierRejectsInvalidHeaders(t *testing.T) {
	t.Parallel()

	_, verifier := newTestPair(t)
	valid := validHeaders(t)

	tests := map[string]struct {
		mutate  func(http.Header)
		wantErr error
	}{
		"missing id": {
			mutate:  func(headers http.Header) { headers.Del(HeaderID) },
			wantErr: ErrMissingHeader,
		},
		"empty id": {
			mutate:  func(headers http.Header) { headers.Set(HeaderID, "") },
			wantErr: ErrMissingHeader,
		},
		"duplicate id": {
			mutate: func(headers http.Header) {
				headers.Add(HeaderID, "msg_other")
			},
			wantErr: ErrInvalidMessageID,
		},
		"invalid id": {
			mutate: func(headers http.Header) {
				headers.Set(HeaderID, "msg.invalid")
			},
			wantErr: ErrInvalidMessageID,
		},
		"missing timestamp": {
			mutate: func(headers http.Header) {
				headers.Del(HeaderTimestamp)
			},
			wantErr: ErrMissingHeader,
		},
		"duplicate timestamp": {
			mutate: func(headers http.Header) {
				headers.Add(HeaderTimestamp, "1700000001")
			},
			wantErr: ErrInvalidTimestamp,
		},
		"invalid timestamp": {
			mutate: func(headers http.Header) {
				headers.Set(HeaderTimestamp, "not-a-time")
			},
			wantErr: ErrInvalidTimestamp,
		},
		"negative timestamp": {
			mutate: func(headers http.Header) {
				headers.Set(HeaderTimestamp, "-1")
			},
			wantErr: ErrInvalidTimestamp,
		},
		"missing signature": {
			mutate: func(headers http.Header) {
				headers.Del(HeaderSignature)
			},
			wantErr: ErrMissingHeader,
		},
		"invalid signature": {
			mutate: func(headers http.Header) {
				headers.Set(HeaderSignature, "v1,invalid")
			},
			wantErr: ErrInvalidSignature,
		},
		"unknown signature version": {
			mutate: func(headers http.Header) {
				headers.Set(HeaderSignature, "v2,unknown")
			},
			wantErr: ErrInvalidSignature,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			headers := valid.Clone()
			test.mutate(headers)

			_, err := verifier.Verify(headers, []byte("payload"))

			assert.ErrorIs(t, err, test.wantErr)
		})
	}
}

func TestVerifierRejectsTampering(t *testing.T) {
	t.Parallel()

	signer, verifier := newTestPair(t)
	payload := []byte("payload")
	headers, err := signer.Sign("msg_tampering", payload)
	require.NoError(t, err)

	tests := map[string]struct {
		headers http.Header
		payload []byte
	}{
		"message id": {
			headers: func() http.Header {
				modified := headers.Clone()
				modified.Set(HeaderID, "msg_other")
				return modified
			}(),
			payload: payload,
		},
		"timestamp": {
			headers: func() http.Header {
				modified := headers.Clone()
				modified.Set(HeaderTimestamp, "1700000001")
				return modified
			}(),
			payload: payload,
		},
		"payload": {
			headers: headers,
			payload: []byte("modified"),
		},
		"signature": {
			headers: func() http.Header {
				modified := headers.Clone()
				modified.Set(
					HeaderSignature,
					"v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
				)
				return modified
			}(),
			payload: payload,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, verifyErr := verifier.Verify(test.headers, test.payload)

			assert.ErrorIs(t, verifyErr, ErrInvalidSignature)
		})
	}
}

func TestVerifierLimitsSignatureParsing(t *testing.T) {
	t.Parallel()

	_, verifier := newTestPair(t)
	headers := validHeaders(t)

	headers.Set(
		HeaderSignature,
		strings.Repeat("v2,unknown ", maxSignatureCount+1),
	)
	_, err := verifier.Verify(headers, []byte("payload"))
	assert.ErrorIs(t, err, ErrInvalidSignature)

	headers.Set(
		HeaderSignature,
		strings.Repeat("x", maxSignatureHeaderBytes+1),
	)
	_, err = verifier.Verify(headers, []byte("payload"))
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestVerifierConcurrentUse(t *testing.T) {
	t.Parallel()

	signer, verifier := newTestPair(t)
	payload := []byte("payload")
	headers, err := signer.Sign("msg_concurrent", payload)
	require.NoError(t, err)

	var wait sync.WaitGroup
	for range 50 {
		wait.Go(func() {
			metadata, verifyErr := verifier.Verify(headers, payload)
			assert.NoError(t, verifyErr)
			assert.Equal(t, "msg_concurrent", metadata.ID)
		})
	}
	wait.Wait()
}

func FuzzSignAndVerify(f *testing.F) {
	f.Add("msg_example", []byte(`{"type":"example.created"}`))
	f.Add("", []byte(nil))

	secret, err := NewSecret(bytesOf(0, 32))
	require.NoError(f, err)
	signer, err := NewSigner(secret)
	require.NoError(f, err)
	verifier, err := NewVerifier(secret)
	require.NoError(f, err)
	signer.now = func() time.Time { return testTime }
	verifier.now = func() time.Time { return testTime }

	f.Fuzz(func(t *testing.T, messageID string, payload []byte) {
		headers, signErr := signer.Sign(messageID, payload)
		if signErr != nil {
			require.ErrorIs(t, signErr, ErrInvalidMessageID)
			return
		}

		metadata, verifyErr := verifier.Verify(headers, payload)
		require.NoError(t, verifyErr)
		assert.Equal(t, messageID, metadata.ID)
	})
}

func FuzzVerifierHeaders(f *testing.F) {
	f.Add("msg_example", "1700000000", "v1,invalid", []byte("payload"))
	f.Add("", "", "", []byte(nil))

	verifier, err := NewVerifier(mustSecret(f, 0))
	require.NoError(f, err)
	verifier.now = func() time.Time { return testTime }

	f.Fuzz(func(
		t *testing.T,
		messageID string,
		timestamp string,
		signature string,
		payload []byte,
	) {
		headers := make(http.Header)
		headers.Set(HeaderID, messageID)
		headers.Set(HeaderTimestamp, timestamp)
		headers.Set(HeaderSignature, signature)

		_, _ = verifier.Verify(headers, payload)
	})
}

func BenchmarkVerifierVerify(b *testing.B) {
	current := mustSecret(b, 0)
	previous := mustSecret(b, 1)
	payload := []byte(`{"type":"example.created","data":{"id":"123"}}`)

	benchmarks := map[string]struct {
		signerOptions   []SignerOption
		verifierOptions []VerifierOption
	}{
		"single secret": {},
		"rotation": {
			signerOptions: []SignerOption{
				WithAdditionalSigningSecrets(previous),
			},
			verifierOptions: []VerifierOption{
				WithAdditionalVerificationSecrets(previous),
			},
		},
	}

	for name, benchmark := range benchmarks {
		b.Run(name, func(b *testing.B) {
			signer, err := NewSigner(current, benchmark.signerOptions...)
			require.NoError(b, err)
			verifier, err := NewVerifier(
				current,
				benchmark.verifierOptions...,
			)
			require.NoError(b, err)
			signer.now = func() time.Time { return testTime }
			verifier.now = func() time.Time { return testTime }
			headers, err := signer.Sign("msg_benchmark", payload)
			require.NoError(b, err)

			b.ReportAllocs()
			for b.Loop() {
				_, err = verifier.Verify(headers, payload)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func newTestPair(t testing.TB) (*Signer, *Verifier) {
	t.Helper()

	secret := mustSecret(t, 0)
	signer, err := NewSigner(secret)
	require.NoError(t, err)
	verifier, err := NewVerifier(secret)
	require.NoError(t, err)
	signer.now = func() time.Time { return testTime }
	verifier.now = func() time.Time { return testTime }
	return signer, verifier
}

func validHeaders(t testing.TB) http.Header {
	t.Helper()

	signer, _ := newTestPair(t)
	headers, err := signer.Sign("msg_valid", []byte("payload"))
	require.NoError(t, err)
	return headers
}
