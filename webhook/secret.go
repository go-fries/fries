package webhook

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	secretPrefix    = "whsec_"
	minSecretLength = 24
	maxSecretLength = 64
)

// Secret contains an HMAC signing key. Its zero value is invalid.
//
// Secret intentionally provides no string or marshaling method that could
// expose its key material.
type Secret struct {
	key []byte
}

// NewSecret creates a Secret from raw key bytes.
//
// The key must contain between 24 and 64 bytes. NewSecret copies key and does
// not retain the caller's slice.
func NewSecret(key []byte) (Secret, error) {
	if err := validateSecretLength(len(key)); err != nil {
		return Secret{}, err
	}

	return Secret{key: bytesClone(key)}, nil
}

// ParseSecret parses a Standard Webhooks secret in whsec_<base64> format.
func ParseSecret(value string) (Secret, error) {
	encoded, ok := strings.CutPrefix(value, secretPrefix)
	if !ok || encoded == "" {
		return Secret{}, fmt.Errorf(
			"%w: expected %s<base64>",
			ErrInvalidSecret,
			secretPrefix,
		)
	}

	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Secret{}, fmt.Errorf(
			"%w: decode base64: %v",
			ErrInvalidSecret,
			err,
		)
	}

	return NewSecret(key)
}

func validateSecret(secret Secret) error {
	return validateSecretLength(len(secret.key))
}

func validateSecretLength(length int) error {
	if length < minSecretLength || length > maxSecretLength {
		return fmt.Errorf(
			"%w: key length must be between %d and %d bytes",
			ErrInvalidSecret,
			minSecretLength,
			maxSecretLength,
		)
	}
	return nil
}

func bytesClone(value []byte) []byte {
	return append([]byte(nil), value...)
}
