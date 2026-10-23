package memory_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
	"github.com/go-fries/fries/notification/v4/database/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRecord() database.Record {
	return database.Record{
		ID: "delivery-42", NotificationID: "notification-42",
		Recipient: notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"},
		Type:      "invoice.paid.v1",
		Data:      json.RawMessage(`{"invoice_id":"INV-42"}`),
		CreatedAt: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC),
	}
}

func TestStoreLifecycleAndDuplicatePreserveReadState(t *testing.T) {
	var store memory.Store
	record := newRecord()
	page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, page.Records)
	require.NoError(t, store.Insert(t.Context(), record))
	count, err := store.CountUnread(t.Context(), record.Recipient)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.NoError(t, store.MarkRead(t.Context(), record.Recipient, record.ID))
	page, err = store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	require.NotNil(t, page.Records[0].ReadAt)
	firstReadAt := *page.Records[0].ReadAt
	require.NoError(t, store.MarkRead(t.Context(), record.Recipient, record.ID))
	duplicate := record
	duplicate.CreatedAt = record.CreatedAt.Add(time.Hour)
	require.NoError(t, store.Insert(t.Context(), duplicate))
	page, err = store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.Equal(t, record.CreatedAt, page.Records[0].CreatedAt)
	assert.Equal(t, firstReadAt, *page.Records[0].ReadAt)
	count, err = store.CountUnread(t.Context(), record.Recipient)
	require.NoError(t, err)
	assert.Zero(t, count)
	page, err = store.List(t.Context(), record.Recipient, database.ListOptions{UnreadOnly: true})
	require.NoError(t, err)
	assert.Empty(t, page.Records)
	require.NoError(t, store.Delete(t.Context(), record.Recipient, record.ID))
	require.ErrorIs(t, store.Delete(t.Context(), record.Recipient, record.ID), database.ErrNotFound)
	require.ErrorIs(t, store.MarkRead(t.Context(), record.Recipient, record.ID), database.ErrNotFound)
}

func TestInsertRejectsConflictingIdentityOrData(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*database.Record)
	}{
		{"tenant", func(r *database.Record) { r.Recipient.TenantID = "other" }},
		{"recipient type", func(r *database.Record) { r.Recipient.Type = "admin" }},
		{"recipient ID", func(r *database.Record) { r.Recipient.ID = "43" }},
		{"notification ID", func(r *database.Record) { r.NotificationID = "another" }},
		{"notification type", func(r *database.Record) { r.Type = "another.v1" }},
		{"data", func(r *database.Record) { r.Data = json.RawMessage(`{"different":true}`) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := memory.New()
			record := newRecord()
			require.NoError(t, store.Insert(t.Context(), record))
			conflict := newRecord()
			tc.mutate(&conflict)
			require.ErrorIs(t, store.Insert(t.Context(), conflict), database.ErrConflict)
			page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
			require.NoError(t, err)
			assert.Equal(t, []database.Record{record}, page.Records)
		})
	}
}

func TestInboxIsScopedByTenantTypeAndID(t *testing.T) {
	store := memory.New()
	record := newRecord()
	require.NoError(t, store.Insert(t.Context(), record))
	for _, other := range []notification.RecipientKey{
		{TenantID: "other", Type: "user", ID: "42"},
		{TenantID: "acme", Type: "admin", ID: "42"},
		{TenantID: "acme", Type: "user", ID: "43"},
	} {
		page, err := store.List(t.Context(), other, database.ListOptions{})
		require.NoError(t, err)
		assert.Empty(t, page.Records)
		count, err := store.CountUnread(t.Context(), other)
		require.NoError(t, err)
		assert.Zero(t, count)
		require.ErrorIs(t, store.MarkRead(t.Context(), other, record.ID), database.ErrNotFound)
		require.ErrorIs(t, store.Delete(t.Context(), other, record.ID), database.ErrNotFound)
	}
	page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.Nil(t, page.Records[0].ReadAt)
}

func TestStoreCopiesMutableRecordFields(t *testing.T) {
	store := memory.New()
	record := newRecord()
	readAt := record.CreatedAt.Add(time.Minute)
	record.ReadAt = &readAt
	expectedReadAt := readAt
	expectedData := string(record.Data)
	require.NoError(t, store.Insert(t.Context(), record))
	record.Data[0] = 'x'
	*record.ReadAt = record.CreatedAt.Add(time.Hour)
	page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.Equal(t, expectedData, string(page.Records[0].Data))
	assert.Equal(t, expectedReadAt, *page.Records[0].ReadAt)
	page.Records[0].Data[0] = 'y'
	*page.Records[0].ReadAt = time.Time{}
	page, err = store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, expectedData, string(page.Records[0].Data))
	assert.Equal(t, expectedReadAt, *page.Records[0].ReadAt)
}

func TestCursorPaginationOrdersTiesAndDoesNotRepeatNewerRecords(t *testing.T) {
	store := memory.New()
	base := newRecord()
	for i, id := range []string{"a", "b", "c", "d", "e", "f"} {
		record := base
		record.ID = id
		record.CreatedAt = base.CreatedAt.Add(time.Duration(i/2) * time.Second)
		require.NoError(t, store.Insert(t.Context(), record))
	}
	options := database.ListOptions{Limit: 2}
	var ids []string
	for pageNumber := 0; ; pageNumber++ {
		require.Less(t, pageNumber, 4, "pagination must make progress")
		page, err := store.List(t.Context(), base.Recipient, options)
		require.NoError(t, err)
		for _, record := range page.Records {
			ids = append(ids, record.ID)
		}
		if page.NextCursor == "" {
			break
		}
		options.Cursor = page.NextCursor
		if pageNumber == 0 {
			newer := base
			newer.ID = "newer"
			newer.CreatedAt = base.CreatedAt.Add(time.Hour)
			require.NoError(t, store.Insert(t.Context(), newer))
		}
	}
	assert.Equal(t, []string{"f", "e", "d", "c", "b", "a"}, ids)
}

func TestUnreadPaginationAndCursorScope(t *testing.T) {
	store := memory.New()
	base := newRecord()
	for _, id := range []string{"a", "b", "c"} {
		record := base
		record.ID = id
		require.NoError(t, store.Insert(t.Context(), record))
	}
	require.NoError(t, store.MarkRead(t.Context(), base.Recipient, "b"))
	page, err := store.List(t.Context(), base.Recipient, database.ListOptions{Limit: 1, UnreadOnly: true})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.Equal(t, "c", page.Records[0].ID)
	require.NotEmpty(t, page.NextCursor)
	options := database.ListOptions{Limit: 1, UnreadOnly: true, Cursor: page.NextCursor}
	second, err := store.List(t.Context(), base.Recipient, options)
	require.NoError(t, err)
	require.Len(t, second.Records, 1)
	assert.Equal(t, "a", second.Records[0].ID)
	assert.Empty(t, second.NextCursor)
	other := base.Recipient
	other.TenantID = "other"
	_, err = store.List(t.Context(), other, options)
	require.ErrorIs(t, err, database.ErrInvalidQuery)
	options.UnreadOnly = false
	_, err = store.List(t.Context(), base.Recipient, options)
	require.ErrorIs(t, err, database.ErrInvalidQuery)
}

func TestListLimitsAndInvalidCursors(t *testing.T) {
	store := memory.New()
	base := newRecord()
	for i := range 25 {
		record := base
		record.ID = strconv.Itoa(i)
		require.NoError(t, store.Insert(t.Context(), record))
	}
	page, err := store.List(t.Context(), base.Recipient, database.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, page.Records, 20)
	assert.NotEmpty(t, page.NextCursor)
	page, err = store.List(t.Context(), base.Recipient, database.ListOptions{Limit: 100})
	require.NoError(t, err)
	assert.Len(t, page.Records, 25)
	for _, limit := range []int{-1, 101} {
		_, err := store.List(t.Context(), base.Recipient, database.ListOptions{Limit: limit})
		require.ErrorIs(t, err, database.ErrInvalidQuery)
	}
	for _, value := range []string{
		"not base64!", base64.RawURLEncoding.EncodeToString([]byte("not json")),
		base64.RawURLEncoding.EncodeToString([]byte(`{}`)),
	} {
		_, err := store.List(t.Context(), base.Recipient, database.ListOptions{Cursor: value})
		require.ErrorIs(t, err, database.ErrInvalidQuery)
	}
}

func TestStoreValidationAndCancellation(t *testing.T) {
	store := memory.New()
	record := newRecord()
	bad := record
	bad.Data = json.RawMessage(`not-json`)
	require.ErrorIs(t, store.Insert(t.Context(), bad), database.ErrInvalidRecord)
	bad = record
	bad.Recipient = notification.RecipientKey{}
	require.ErrorIs(t, store.Insert(t.Context(), bad), notification.ErrInvalidRecipient)
	_, err := store.List(t.Context(), bad.Recipient, database.ListOptions{})
	require.ErrorIs(t, err, notification.ErrInvalidRecipient)
	_, err = store.CountUnread(t.Context(), bad.Recipient)
	require.ErrorIs(t, err, notification.ErrInvalidRecipient)
	require.ErrorIs(t, store.MarkRead(t.Context(), bad.Recipient, record.ID), notification.ErrInvalidRecipient)
	require.ErrorIs(t, store.Delete(t.Context(), bad.Recipient, record.ID), notification.ErrInvalidRecipient)
	require.ErrorIs(t, store.MarkRead(t.Context(), record.Recipient, ""), database.ErrInvalidQuery)
	require.ErrorIs(t, store.Delete(t.Context(), record.Recipient, " "), database.ErrInvalidQuery)
	require.NoError(t, store.Insert(t.Context(), record))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, store.Insert(ctx, record), context.Canceled)
	_, err = store.List(ctx, record.Recipient, database.ListOptions{})
	require.ErrorIs(t, err, context.Canceled)
	_, err = store.CountUnread(ctx, record.Recipient)
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, store.MarkRead(ctx, record.Recipient, record.ID), context.Canceled)
	require.ErrorIs(t, store.Delete(ctx, record.Recipient, record.ID), context.Canceled)
	page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.Nil(t, page.Records[0].ReadAt)
}

func TestStoreConcurrentSafe(t *testing.T) {
	store := memory.New()
	record := newRecord()
	const count = 64
	errs := make(chan error, count*5)
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			errs <- store.Insert(t.Context(), record)
			errs <- store.MarkRead(t.Context(), record.Recipient, record.ID)
			_, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
			errs <- err
			_, err = store.CountUnread(t.Context(), record.Recipient)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	page, err := store.List(t.Context(), record.Recipient, database.ListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Records, 1)
	assert.NotNil(t, page.Records[0].ReadAt)
	countUnread, err := store.CountUnread(t.Context(), record.Recipient)
	require.NoError(t, err)
	assert.Zero(t, countUnread)
}

var (
	_ database.Writer = (*memory.Store)(nil)
	_ database.Inbox  = (*memory.Store)(nil)
)
