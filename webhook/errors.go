package webhook

import "errors"

var (
	// ErrInvalidSecret indicates that a signing secret is malformed or does
	// not meet the required security length.
	ErrInvalidSecret = errors.New("webhook: invalid secret")
	// ErrInvalidTolerance indicates that a verification tolerance is not
	// positive.
	ErrInvalidTolerance = errors.New("webhook: invalid tolerance")
	// ErrInvalidMessageID indicates that a message ID cannot be signed or
	// verified.
	ErrInvalidMessageID = errors.New("webhook: invalid message id")
	// ErrMissingHeader indicates that a required Webhook header is absent.
	ErrMissingHeader = errors.New("webhook: missing required header")
	// ErrInvalidTimestamp indicates that webhook-timestamp is malformed.
	ErrInvalidTimestamp = errors.New("webhook: invalid timestamp")
	// ErrTimestampOutsideTolerance indicates that the message timestamp is too
	// old or too far in the future.
	ErrTimestampOutsideTolerance = errors.New(
		"webhook: timestamp outside tolerance",
	)
	// ErrInvalidSignature indicates that no supported signature matches the
	// message.
	ErrInvalidSignature = errors.New("webhook: invalid signature")
)
