package notification

import (
	"strings"
	"unicode/utf8"
)

// Recipient identifies a notification recipient. Channel-specific routing is
// provided through additional interfaces owned by the channel packages.
// An anonymous recipient returns a zero key and cannot use a database inbox.
// Nonzero keys must be valid; Manager rejects partially populated keys.
// Recipients must not be mutated while a Send is in progress.
type Recipient interface {
	NotificationKey() RecipientKey
}

// RecipientKey scopes an inbox to a tenant, entity type, and business ID.
// TenantID may be empty for single-tenant applications. Type and ID are required
// for persistent inbox operations. Keys are opaque, case-sensitive UTF-8 text.
type RecipientKey struct {
	TenantID string
	Type     string
	ID       string
}

// Valid reports whether the key can identify a persistent inbox recipient.
func (k RecipientKey) Valid() bool {
	return strings.TrimSpace(k.Type) != "" && strings.TrimSpace(k.ID) != "" &&
		utf8.ValidString(k.TenantID) && utf8.ValidString(k.Type) && utf8.ValidString(k.ID)
}
