package memory

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
)

// Store is an in-memory Writer and Inbox. It copies record data on input and
// output and atomically enforces delivery-ID uniqueness. Its zero value is ready
// to use. A Store must not be copied after first use.
type Store struct {
	mu      sync.RWMutex
	records map[string]database.Record
}

// New creates an empty Store.
func New() *Store {
	return &Store{}
}

// Insert implements database.Writer, preserving timestamps on duplicate inserts.
func (s *Store) Insert(ctx context.Context, record database.Record) error {
	if err := checkScope(ctx, record.Recipient); err != nil {
		return err
	}
	if err := record.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if existing, ok := s.records[record.ID]; ok {
		if existing.Recipient != record.Recipient || existing.NotificationID != record.NotificationID ||
			existing.Type != record.Type || !bytes.Equal(existing.Data, record.Data) {
			return database.ErrConflict
		}
		return nil
	}
	if s.records == nil {
		s.records = make(map[string]database.Record)
	}
	s.records[record.ID] = clone(record)
	return nil
}

// List implements database.Inbox. Records are ordered newest first, with
// descending IDs breaking timestamp ties. Invalid or differently scoped cursors
// return database.ErrInvalidQuery; cursors do not grant access to any records.
func (s *Store) List(ctx context.Context, to notification.RecipientKey, options database.ListOptions) (database.Page, error) {
	if err := checkScope(ctx, to); err != nil {
		return database.Page{}, err
	}
	limit := options.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return database.Page{}, database.ErrInvalidQuery
	}
	anchor, err := decodeCursor(options.Cursor, to, options.UnreadOnly)
	if err != nil {
		return database.Page{}, err
	}
	records, err := s.snapshot(ctx, to, options.UnreadOnly)
	if err != nil {
		return database.Page{}, err
	}
	slices.SortFunc(records, func(a, b database.Record) int {
		if order := b.CreatedAt.Compare(a.CreatedAt); order != 0 {
			return order
		}
		return strings.Compare(b.ID, a.ID)
	})
	page := database.Page{Records: make([]database.Record, 0, min(limit, len(records)))}
	for _, record := range records {
		if options.Cursor != "" && !olderThan(record, anchor) {
			continue
		}
		if len(page.Records) == limit {
			page.NextCursor = encodeCursor(page.Records[len(page.Records)-1], options.UnreadOnly)
			break
		}
		page.Records = append(page.Records, record)
	}
	if err := ctx.Err(); err != nil {
		return database.Page{}, err
	}
	return page, nil
}

// CountUnread counts unread records belonging to the supplied recipient.
func (s *Store) CountUnread(ctx context.Context, to notification.RecipientKey) (int, error) {
	if err := checkScope(ctx, to); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	count := 0
	for _, record := range s.records {
		if record.Recipient == to && record.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

// MarkRead sets the first read timestamp and preserves it on later calls.
func (s *Store) MarkRead(ctx context.Context, to notification.RecipientKey, id string) error {
	if err := checkRecordQuery(ctx, to, id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	record, exists := s.records[id]
	if !exists || record.Recipient != to {
		return database.ErrNotFound
	}
	if record.ReadAt == nil {
		now := time.Now().UTC()
		record.ReadAt = &now
		s.records[id] = record
	}
	return nil
}

// Delete removes a record only within the supplied recipient's scope.
func (s *Store) Delete(ctx context.Context, to notification.RecipientKey, id string) error {
	if err := checkRecordQuery(ctx, to, id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	record, exists := s.records[id]
	if !exists || record.Recipient != to {
		return database.ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func (s *Store) snapshot(ctx context.Context, to notification.RecipientKey, unreadOnly bool) ([]database.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	records := make([]database.Record, 0)
	for _, record := range s.records {
		if record.Recipient == to && (!unreadOnly || record.ReadAt == nil) {
			records = append(records, clone(record))
		}
	}
	return records, nil
}

func clone(record database.Record) database.Record {
	record.Data = slices.Clone(record.Data)
	if record.ReadAt != nil {
		readAt := *record.ReadAt
		record.ReadAt = &readAt
	}
	return record
}

func checkScope(ctx context.Context, to notification.RecipientKey) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !to.Valid() {
		return notification.ErrInvalidRecipient
	}
	return nil
}

func checkRecordQuery(ctx context.Context, to notification.RecipientKey, id string) error {
	if err := checkScope(ctx, to); err != nil {
		return err
	}
	if strings.TrimSpace(id) == "" {
		return database.ErrInvalidQuery
	}
	return nil
}
