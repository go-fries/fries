package webhook

import (
	"crypto/hmac"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	maxSignatureHeaderBytes = 16 << 10
	maxSignatureCount       = 32
)

// Metadata contains authenticated webhook message metadata.
type Metadata struct {
	// ID is the signed webhook message ID.
	ID string
	// Timestamp is the signed message timestamp.
	Timestamp time.Time
}

// Verifier authenticates Standard Webhooks HMAC-SHA256 signatures.
//
// A Verifier is safe for concurrent use.
type Verifier struct {
	secrets   [][]byte
	tolerance time.Duration
	now       func() time.Time
}

// NewVerifier creates a Verifier with secret as its primary verification
// secret.
func NewVerifier(
	secret Secret,
	options ...VerifierOption,
) (*Verifier, error) {
	c, err := newVerifierConfig(secret, options...)
	if err != nil {
		return nil, err
	}

	return &Verifier{
		secrets:   c.secrets,
		tolerance: c.tolerance,
		now:       time.Now,
	}, nil
}

// Verify authenticates the Webhook headers and exact payload bytes.
//
// On success, Metadata contains the signed message ID and timestamp. Verify
// does not decode payload or prevent the same valid message from being
// processed more than once.
func (v *Verifier) Verify(
	headers http.Header,
	payload []byte,
) (Metadata, error) {
	messageID, err := requiredSingleHeader(
		headers,
		HeaderID,
		ErrInvalidMessageID,
	)
	if err != nil {
		return Metadata{}, err
	}
	if !validMessageID(messageID) {
		return Metadata{}, ErrInvalidMessageID
	}

	timestampValue, err := requiredSingleHeader(
		headers,
		HeaderTimestamp,
		ErrInvalidTimestamp,
	)
	if err != nil {
		return Metadata{}, err
	}
	unixTimestamp, err := strconv.ParseInt(timestampValue, 10, 64)
	if err != nil || unixTimestamp < 0 {
		return Metadata{}, ErrInvalidTimestamp
	}

	timestamp := time.Unix(unixTimestamp, 0).UTC()
	difference := v.now().Sub(timestamp)
	if difference > v.tolerance || difference < -v.tolerance {
		return Metadata{}, ErrTimestampOutsideTolerance
	}

	signatures, err := parseSignatures(headers)
	if err != nil {
		return Metadata{}, err
	}

	matched := false
	for _, secret := range v.secrets {
		expected := signDigest(
			secret,
			messageID,
			timestampValue,
			payload,
		)
		for _, signature := range signatures {
			equal := hmac.Equal(expected, signature)
			matched = equal || matched
		}
	}
	if !matched {
		return Metadata{}, ErrInvalidSignature
	}

	return Metadata{
		ID:        messageID,
		Timestamp: timestamp,
	}, nil
}

func requiredSingleHeader(
	headers http.Header,
	name string,
	invalidError error,
) (string, error) {
	values := headers.Values(name)
	if len(values) == 0 || len(values) == 1 && values[0] == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingHeader, name)
	}
	if len(values) != 1 {
		return "", fmt.Errorf("%w: duplicate %s", invalidError, name)
	}
	return values[0], nil
}

func parseSignatures(headers http.Header) ([][]byte, error) {
	values := headers.Values(HeaderSignature)
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrMissingHeader, HeaderSignature)
	}

	totalBytes := 0
	tokenCount := 0
	signatures := make([][]byte, 0, len(values))
	for _, value := range values {
		totalBytes += len(value)
		if totalBytes > maxSignatureHeaderBytes {
			return nil, ErrInvalidSignature
		}

		for token := range strings.FieldsSeq(value) {
			tokenCount++
			if tokenCount > maxSignatureCount {
				return nil, ErrInvalidSignature
			}

			version, encoded, ok := strings.Cut(token, ",")
			if !ok || version != signatureVersion {
				continue
			}
			signature, err := base64.StdEncoding.DecodeString(encoded)
			if err != nil || len(signature) != hmacSignatureSize {
				continue
			}
			signatures = append(signatures, signature)
		}
	}

	if len(signatures) == 0 {
		return nil, ErrInvalidSignature
	}
	return signatures, nil
}

const hmacSignatureSize = 32
