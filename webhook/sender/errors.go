package sender

import "errors"

var (
	// ErrInvalidContext indicates that a required context is nil.
	ErrInvalidContext = errors.New("webhook/sender: invalid context")
	// ErrInvalidEndpoint indicates that an endpoint URL is malformed or
	// unsupported.
	ErrInvalidEndpoint = errors.New("webhook/sender: invalid endpoint")
	// ErrInsecureEndpoint indicates that an HTTP endpoint was supplied without
	// explicitly enabling insecure HTTP.
	ErrInsecureEndpoint = errors.New("webhook/sender: insecure endpoint")
	// ErrNilSigner indicates that a nil webhook Signer was supplied to [New].
	ErrNilSigner = errors.New("webhook/sender: nil signer")
)
