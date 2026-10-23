package memory

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
)

type cursor struct {
	Recipient  notification.RecipientKey `json:"recipient"`
	CreatedAt  time.Time                 `json:"created_at"`
	ID         string                    `json:"id"`
	UnreadOnly bool                      `json:"unread_only"`
}

func decodeCursor(value string, to notification.RecipientKey, unreadOnly bool) (cursor, error) {
	if value == "" {
		return cursor{}, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, database.ErrInvalidQuery
	}
	var c cursor
	if err := json.Unmarshal(data, &c); err != nil {
		return cursor{}, database.ErrInvalidQuery
	}
	if c.Recipient != to || c.UnreadOnly != unreadOnly || c.CreatedAt.IsZero() || c.ID == "" {
		return cursor{}, database.ErrInvalidQuery
	}
	return c, nil
}

func encodeCursor(record database.Record, unreadOnly bool) string {
	// Record.Validate guarantees a valid timestamp; encoding this struct cannot fail.
	data, _ := json.Marshal(cursor{
		Recipient:  record.Recipient,
		CreatedAt:  record.CreatedAt,
		ID:         record.ID,
		UnreadOnly: unreadOnly,
	})
	return base64.RawURLEncoding.EncodeToString(data)
}

func olderThan(record database.Record, anchor cursor) bool {
	return record.CreatedAt.Before(anchor.CreatedAt) ||
		(record.CreatedAt.Equal(anchor.CreatedAt) && record.ID < anchor.ID)
}
