package webhook

import (
	"fmt"
	"time"
)

const defaultTolerance = 5 * time.Minute

type signerConfig struct {
	secrets [][]byte
}

// SignerOption configures a [Signer].
type SignerOption interface {
	applySigner(*signerConfig) error
}

type signerOptionFunc func(*signerConfig) error

func (f signerOptionFunc) applySigner(c *signerConfig) error {
	return f(c)
}

// WithAdditionalSigningSecrets adds secrets whose signatures are emitted
// after the primary signature. This supports zero-downtime secret rotation.
func WithAdditionalSigningSecrets(secrets ...Secret) SignerOption {
	return signerOptionFunc(func(c *signerConfig) error {
		for _, secret := range secrets {
			if err := validateSecret(secret); err != nil {
				return err
			}
			c.secrets = append(c.secrets, bytesClone(secret.key))
		}
		return nil
	})
}

func newSignerConfig(
	secret Secret,
	options ...SignerOption,
) (signerConfig, error) {
	if err := validateSecret(secret); err != nil {
		return signerConfig{}, err
	}

	c := signerConfig{
		secrets: [][]byte{bytesClone(secret.key)},
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option.applySigner(&c); err != nil {
			return signerConfig{}, err
		}
	}
	return c, nil
}

type verifierConfig struct {
	secrets   [][]byte
	tolerance time.Duration
}

// VerifierOption configures a [Verifier].
type VerifierOption interface {
	applyVerifier(*verifierConfig) error
}

type verifierOptionFunc func(*verifierConfig) error

func (f verifierOptionFunc) applyVerifier(c *verifierConfig) error {
	return f(c)
}

// WithTolerance sets the maximum allowed difference between the message
// timestamp and the current time.
//
// The tolerance must be positive.
func WithTolerance(tolerance time.Duration) VerifierOption {
	return verifierOptionFunc(func(c *verifierConfig) error {
		if tolerance <= 0 {
			return fmt.Errorf(
				"%w: must be greater than zero",
				ErrInvalidTolerance,
			)
		}
		c.tolerance = tolerance
		return nil
	})
}

// WithAdditionalVerificationSecrets adds secrets accepted in addition to the
// primary secret. This supports zero-downtime secret rotation.
func WithAdditionalVerificationSecrets(
	secrets ...Secret,
) VerifierOption {
	return verifierOptionFunc(func(c *verifierConfig) error {
		for _, secret := range secrets {
			if err := validateSecret(secret); err != nil {
				return err
			}
			c.secrets = append(c.secrets, bytesClone(secret.key))
		}
		return nil
	})
}

func newVerifierConfig(
	secret Secret,
	options ...VerifierOption,
) (verifierConfig, error) {
	if err := validateSecret(secret); err != nil {
		return verifierConfig{}, err
	}

	c := verifierConfig{
		secrets:   [][]byte{bytesClone(secret.key)},
		tolerance: defaultTolerance,
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option.applyVerifier(&c); err != nil {
			return verifierConfig{}, err
		}
	}
	return c, nil
}
