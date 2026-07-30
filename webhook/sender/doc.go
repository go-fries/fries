// Package sender delivers signed webhook messages over HTTP.
//
// A [Sender] binds one endpoint to one webhook Signer and performs one HTTP
// request per [Sender.Send] call. It does not persist messages, retry failed
// deliveries, or classify non-success responses as errors.
package sender
