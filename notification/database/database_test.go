package database_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-fries/fries/notification/v4"
	"github.com/go-fries/fries/notification/v4/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recipient struct{ key notification.RecipientKey }

func (r recipient) NotificationKey() notification.RecipientKey { return r.key }

type basicNotice struct{}

func (basicNotice) Type() string { return "invoice.paid.v1" }
func (basicNotice) Via(context.Context, notification.Recipient) ([]string, error) {
	return []string{database.Name}, nil
}

type notice struct {
	basicNotice
	build func(context.Context, notification.Recipient) (any, error)
}

func (n notice) ToDatabase(ctx context.Context, to notification.Recipient) (any, error) {
	if n.build != nil {
		return n.build(ctx, to)
	}
	return struct {
		InvoiceID string `json:"invoice_id"`
	}{InvoiceID: "INV-42"}, nil
}

type writerFunc func(context.Context, database.Record) error

func (f writerFunc) Insert(ctx context.Context, record database.Record) error { return f(ctx, record) }

func validRecord() database.Record {
	return database.Record{
		ID: "delivery-42", NotificationID: "notification-42",
		Recipient: notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"},
		Type:      "invoice.paid.v1",
		Data:      json.RawMessage(`{"invoice_id":"INV-42"}`),
		CreatedAt: time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC),
	}
}

func TestDatabaseChannelBuildsTypedSnapshot(t *testing.T) {
	var records []database.Record
	channel, err := database.NewChannel(writerFunc(func(_ context.Context, record database.Record) error {
		records = append(records, record)
		return nil
	}))
	require.NoError(t, err)
	manager, err := notification.New(notification.WithChannel(database.Name, channel))
	require.NoError(t, err)
	to := recipient{key: validRecord().Recipient}
	result, err := manager.Send(t.Context(), to, notice{}, notification.WithID("notification-42"))
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, result.Deliveries[0].ID, records[0].ID)
	assert.Equal(t, "notification-42", records[0].NotificationID)
	assert.Equal(t, to.key, records[0].Recipient)
	assert.Equal(t, "invoice.paid.v1", records[0].Type)
	assert.JSONEq(t, `{"invoice_id":"INV-42"}`, string(records[0].Data))
	assert.Nil(t, records[0].ReadAt)
	assert.False(t, records[0].CreatedAt.IsZero())
}

func TestDatabaseChannelFailuresDoNotWrite(t *testing.T) {
	buildErr := errors.New("missing invoice")
	valid := validRecord()
	var nilNotice *notice
	var nilRecipient *recipient
	for _, tc := range []struct {
		name string
		n    notification.Notification
		to   notification.Recipient
		id   string
		want error
	}{
		{"unsupported", basicNotice{}, recipient{valid.Recipient}, valid.ID, notification.ErrUnsupportedNotification},
		{"typed nil notification", nilNotice, recipient{valid.Recipient}, valid.ID, notification.ErrUnsupportedNotification},
		{"anonymous", notice{}, recipient{}, valid.ID, notification.ErrInvalidRecipient},
		{"nil recipient", notice{}, nil, valid.ID, notification.ErrInvalidRecipient},
		{"typed nil recipient", notice{}, nilRecipient, valid.ID, notification.ErrInvalidRecipient},
		{"invalid delivery", notice{}, recipient{valid.Recipient}, "", database.ErrInvalidRecord},
		{"builder error", notice{build: func(context.Context, notification.Recipient) (any, error) {
			return nil, buildErr
		}}, recipient{valid.Recipient}, valid.ID, buildErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			channel, err := database.NewChannel(writerFunc(func(context.Context, database.Record) error {
				calls++
				return nil
			}))
			require.NoError(t, err)
			err = channel.Send(t.Context(), notification.Delivery{
				ID: tc.id, NotificationID: valid.NotificationID, CreatedAt: valid.CreatedAt,
				Notification: tc.n, Recipient: tc.to,
			})
			require.ErrorIs(t, err, tc.want)
			assert.Zero(t, calls)
		})
	}
}

func TestDatabaseChannelEncodingAndWriterErrors(t *testing.T) {
	valid := validRecord()
	writerErr := errors.New("store unavailable")
	calls := 0
	channel, err := database.NewChannel(writerFunc(func(context.Context, database.Record) error {
		calls++
		return writerErr
	}))
	require.NoError(t, err)
	delivery := notification.Delivery{
		ID: valid.ID, NotificationID: valid.NotificationID, CreatedAt: valid.CreatedAt,
		Recipient: recipient{valid.Recipient}, Notification: notice{build: func(context.Context, notification.Recipient) (any, error) {
			return make(chan int), nil
		}},
	}
	err = channel.Send(t.Context(), delivery)
	var encodingErr *json.UnsupportedTypeError
	require.ErrorAs(t, err, &encodingErr)
	assert.Zero(t, calls)
	delivery.Notification = notice{}
	require.ErrorIs(t, channel.Send(t.Context(), delivery), writerErr)
	assert.Equal(t, 1, calls)
}

func TestDatabaseChannelCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	channel, err := database.NewChannel(writerFunc(func(context.Context, database.Record) error {
		calls++
		return nil
	}))
	require.NoError(t, err)
	valid := validRecord()
	delivery := notification.Delivery{
		ID: valid.ID, NotificationID: valid.NotificationID, CreatedAt: valid.CreatedAt,
		Recipient: recipient{valid.Recipient}, Notification: notice{build: func(context.Context, notification.Recipient) (any, error) {
			cancel()
			return "data", nil
		}},
	}
	require.ErrorIs(t, channel.Send(ctx, delivery), context.Canceled)
	require.ErrorIs(t, channel.Send(ctx, delivery), context.Canceled)
	assert.Zero(t, calls)
}

func TestDatabaseConstruction(t *testing.T) {
	var typedNil writerFunc
	for _, writer := range []database.Writer{nil, typedNil} {
		channel, err := database.NewChannel(writer)
		require.ErrorIs(t, err, notification.ErrInvalidConfig)
		assert.Nil(t, channel)
	}
}

func TestRecordValidation(t *testing.T) {
	require.NoError(t, validRecord().Validate())
	for _, mutate := range []func(*database.Record){
		func(r *database.Record) { r.ID = "" },
		func(r *database.Record) { r.NotificationID = " " },
		func(r *database.Record) { r.Type = "" },
		func(r *database.Record) { r.ID = "\xff" },
		func(r *database.Record) { r.NotificationID = "\xff" },
		func(r *database.Record) { r.Type = "\xff" },
		func(r *database.Record) { r.CreatedAt = time.Time{} },
		func(r *database.Record) { r.CreatedAt = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(r *database.Record) { r.Data = json.RawMessage(`{invalid`) },
		func(r *database.Record) { r.Data = nil },
	} {
		record := validRecord()
		mutate(&record)
		require.ErrorIs(t, record.Validate(), database.ErrInvalidRecord)
	}
	record := validRecord()
	record.Recipient.ID = ""
	require.ErrorIs(t, record.Validate(), notification.ErrInvalidRecipient)
}
