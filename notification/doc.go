// Package notification delivers business notifications through explicitly
// configured channels. A notification selects its channels with Via; each
// channel owns its message and recipient-routing contracts.
//
// Manager.Send is synchronous and reports each channel's outcome. It does not
// retry, queue work, recover panics, or guarantee exactly-once delivery.
package notification
