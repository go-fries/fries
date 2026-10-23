package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/internal/nilvalue"
)

// Name is the conventional database channel name.
const Name = "database"

var (
	// ErrInvalidRecord indicates missing identity, time or valid JSON data.
	ErrInvalidRecord = errors.New("notification/database: invalid record")
	// ErrConflict indicates a delivery ID was reused for different immutable data.
	ErrConflict = errors.New("notification/database: conflicting delivery")
	// ErrNotFound indicates no record exists in the supplied recipient's scope.
	ErrNotFound = errors.New("notification/database: record not found")
	// ErrInvalidQuery indicates an invalid page size, cursor or record ID.
	ErrInvalidQuery = errors.New("notification/database: invalid query")
)

// Notification builds the JSON-encodable inbox representation. It may return a
// typed application struct; encoding is owned by the database channel.
type Notification interface {
	ToDatabase(ctx context.Context, to notification.Recipient) (any, error)
}

// Record is a stored inbox notification. ID is the channel Delivery.ID.
// Data contains a display snapshot, not a delivery log for other channels.
type Record struct {
	ID             string
	NotificationID string
	Recipient      notification.RecipientKey
	Type           string
	Data           json.RawMessage
	ReadAt         *time.Time
	CreatedAt      time.Time
}

// Validate checks the record's identity, timestamp and JSON representation.
// A recipient may omit TenantID, but must have a Type and ID.
func (r Record) Validate() error {
	if !r.Recipient.Valid() {
		return notification.ErrInvalidRecipient
	}
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.NotificationID) == "" ||
		strings.TrimSpace(r.Type) == "" || r.CreatedAt.IsZero() || !json.Valid(r.Data) ||
		!utf8.ValidString(r.ID) || !utf8.ValidString(r.NotificationID) || !utf8.ValidString(r.Type) {
		return ErrInvalidRecord
	}
	if _, err := r.CreatedAt.MarshalJSON(); err != nil {
		return ErrInvalidRecord
	}
	return nil
}

// Writer inserts inbox records. Repeating the same ID and immutable data must
// succeed without changing the existing creation/read timestamps. Reusing an ID
// with a different recipient, notification ID, type or JSON bytes must return
// ErrConflict. Implementations must atomically enforce this contract.
type Writer interface {
	Insert(ctx context.Context, record Record) error
}

// Inbox describes recipient-scoped inbox operations. Every operation requires
// a valid recipient key. An absent record and a record belonging to another
// recipient both return ErrNotFound from MarkRead and Delete.
type Inbox interface {
	List(ctx context.Context, to notification.RecipientKey, options ListOptions) (Page, error)
	CountUnread(ctx context.Context, to notification.RecipientKey) (int, error)
	MarkRead(ctx context.Context, to notification.RecipientKey, id string) error
	Delete(ctx context.Context, to notification.RecipientKey, id string) error
}

// ListOptions configures descending (CreatedAt, ID) cursor pagination. Limit
// defaults to 20 and must be between 1 and 100 when supplied. Cursor is opaque
// and must be reused only with the same recipient and UnreadOnly filter.
type ListOptions struct {
	Limit      int
	Cursor     string
	UnreadOnly bool
}

// Page contains a recipient's records and a cursor for the next page, if any.
// Pages are not a transactional snapshot across separate List calls.
type Page struct {
	Records    []Record
	NextCursor string
}

// Channel inserts an inbox representation using an injected Writer.
type Channel struct {
	writer Writer
}

// NewChannel creates a database channel. Nil writers, including typed nils,
// return notification.ErrInvalidConfig. The caller owns the writer's resources.
func NewChannel(writer Writer) (*Channel, error) {
	if nilvalue.IsNil(writer) {
		return nil, fmt.Errorf("%w: database writer is nil", notification.ErrInvalidConfig)
	}
	return &Channel{writer: writer}, nil
}

// Send builds a JSON display snapshot and inserts one recipient-scoped record.
// Anonymous recipients cannot use this channel.
func (c *Channel) Send(ctx context.Context, d notification.Delivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	builder, ok := d.Notification.(Notification)
	if !ok || nilvalue.IsNil(builder) {
		return fmt.Errorf("%w: database needs ToDatabase", notification.ErrUnsupportedNotification)
	}
	if nilvalue.IsNil(d.Recipient) {
		return notification.ErrInvalidRecipient
	}
	key := d.Recipient.NotificationKey()
	if !key.Valid() {
		return notification.ErrInvalidRecipient
	}
	data, err := builder.ToDatabase(ctx, d.Recipient)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("notification/database: encode data: %w", err)
	}
	record := Record{
		ID:             d.ID,
		NotificationID: d.NotificationID,
		Recipient:      key,
		Type:           d.Notification.Type(),
		Data:           payload,
		CreatedAt:      d.CreatedAt,
	}
	if err := record.Validate(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.writer.Insert(ctx, record)
}

var _ notification.Channel = (*Channel)(nil)
