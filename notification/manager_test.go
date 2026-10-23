package notification_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-fries/fries/notification/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recipient struct {
	key notification.RecipientKey
}

func (r recipient) NotificationKey() notification.RecipientKey { return r.key }

type notice struct {
	kind     string
	channels []string
	via      func(context.Context, notification.Recipient) ([]string, error)
}

func (n notice) Type() string { return n.kind }

func (n notice) Via(ctx context.Context, to notification.Recipient) ([]string, error) {
	if n.via != nil {
		return n.via(ctx, to)
	}
	return n.channels, nil
}

func newNotice(channels ...string) notice {
	return notice{kind: "invoice.paid.v1", channels: channels}
}

type guardedNotice struct {
	notice
	guard func(context.Context, notification.Recipient, string) (bool, error)
}

func (n guardedNotice) ShouldSend(ctx context.Context, to notification.Recipient, channel string) (bool, error) {
	return n.guard(ctx, to, channel)
}

func TestNewValidatesChannels(t *testing.T) {
	channel := notification.ChannelFunc(func(context.Context, notification.Delivery) error { return nil })
	var typedNil notification.ChannelFunc
	for _, tc := range []struct {
		name    string
		options []notification.Option
	}{
		{"empty name", []notification.Option{notification.WithChannel("", channel)}},
		{"blank name", []notification.Option{notification.WithChannel(" \t", channel)}},
		{"nil channel", []notification.Option{notification.WithChannel("mail", nil)}},
		{"typed nil", []notification.Option{notification.WithChannel("mail", typedNil)}},
		{"duplicate", []notification.Option{notification.WithChannel("mail", channel), notification.WithChannel("mail", channel)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, err := notification.New(tc.options...)
			require.ErrorIs(t, err, notification.ErrInvalidConfig)
			assert.Nil(t, manager)
		})
	}
	manager, err := notification.New(nil, notification.WithChannel("mail", channel))
	require.NoError(t, err)
	assert.NotNil(t, manager)
}

func TestSendContinuesAfterFailuresAndDeduplicates(t *testing.T) {
	wantErr := errors.New("mail unavailable")
	var deliveries []notification.Delivery
	channel := notification.ChannelFunc(func(_ context.Context, d notification.Delivery) error {
		deliveries = append(deliveries, d)
		if d.Channel == "mail" {
			return wantErr
		}
		return nil
	})
	manager, err := notification.New(
		notification.WithChannel("mail", channel),
		notification.WithChannel("database", channel),
	)
	require.NoError(t, err)
	to := recipient{key: notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"}}
	n := newNotice("mail", "mail", "database")
	result, err := manager.Send(t.Context(), to, n, nil, notification.WithID("invoice-123"))
	require.ErrorIs(t, err, wantErr)
	var deliveryErr *notification.DeliveryError
	require.ErrorAs(t, err, &deliveryErr)
	assert.Equal(t, "mail", deliveryErr.Channel)
	assert.Equal(t, "invoice-123", result.NotificationID)
	require.Len(t, result.Deliveries, 2)
	assert.Equal(t, notification.StatusFailed, result.Deliveries[0].Status)
	assert.Equal(t, notification.StatusSent, result.Deliveries[1].Status)
	assert.Nil(t, result.Deliveries[1].Err)
	assert.Equal(t, result.Deliveries[0].ID, deliveryErr.ID)
	assert.Contains(t, deliveryErr.Error(), "mail unavailable")
	require.Len(t, deliveries, 2)
	assert.Equal(t, []string{"mail", "database"}, []string{deliveries[0].Channel, deliveries[1].Channel})
	assert.Equal(t, to, deliveries[0].Recipient)
	assert.Equal(t, n, deliveries[0].Notification)
	assert.Equal(t, result.NotificationID, deliveries[0].NotificationID)
	assert.False(t, deliveries[0].CreatedAt.IsZero())
	assert.Equal(t, deliveries[0].CreatedAt, deliveries[1].CreatedAt)
	assert.NotEqual(t, deliveries[0].ID, deliveries[1].ID)
}

func TestSendPreflightsBeforeSideEffects(t *testing.T) {
	viaErr := errors.New("preferences unavailable")
	var nilNotice *notice
	var nilRecipient *recipient
	for _, tc := range []struct {
		name    string
		to      notification.Recipient
		n       notification.Notification
		options []notification.SendOption
		want    error
	}{
		{"nil recipient", nil, newNotice("mail"), nil, notification.ErrInvalidRecipient},
		{"typed nil recipient", nilRecipient, newNotice("mail"), nil, notification.ErrInvalidRecipient},
		{"partial recipient key", recipient{key: notification.RecipientKey{Type: "user"}}, newNotice("mail"), nil, notification.ErrInvalidRecipient},
		{"invalid recipient text", recipient{key: notification.RecipientKey{Type: "user", ID: "\xff"}}, newNotice("mail"), nil, notification.ErrInvalidRecipient},
		{"nil notification", recipient{}, nil, nil, notification.ErrInvalidNotification},
		{"typed nil notification", recipient{}, nilNotice, nil, notification.ErrInvalidNotification},
		{"empty type", recipient{}, notice{}, nil, notification.ErrInvalidNotification},
		{"blank type", recipient{}, notice{kind: "  "}, nil, notification.ErrInvalidNotification},
		{"empty ID", recipient{}, newNotice("mail"), []notification.SendOption{notification.WithID("")}, notification.ErrInvalidID},
		{"blank ID", recipient{}, newNotice("mail"), []notification.SendOption{notification.WithID("\t")}, notification.ErrInvalidID},
		{"unknown channel", recipient{}, newNotice("mail", "missing"), nil, notification.ErrUnknownChannel},
		{"empty channel", recipient{}, newNotice("mail", ""), nil, notification.ErrUnknownChannel},
		{"via error", recipient{}, notice{kind: "test", via: func(context.Context, notification.Recipient) ([]string, error) {
			return nil, viaErr
		}}, nil, viaErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			manager, err := notification.New(notification.WithChannel("mail", notification.ChannelFunc(
				func(context.Context, notification.Delivery) error { calls++; return nil },
			)))
			require.NoError(t, err)
			result, err := manager.Send(t.Context(), tc.to, tc.n, tc.options...)
			require.ErrorIs(t, err, tc.want)
			assert.Zero(t, result)
			assert.Zero(t, calls)
		})
	}
}

func TestSendGuardSkipsAndFailsIndependently(t *testing.T) {
	guardErr := errors.New("cannot read preferences")
	var sent []string
	channel := notification.ChannelFunc(func(_ context.Context, d notification.Delivery) error {
		sent = append(sent, d.Channel)
		return nil
	})
	manager, err := notification.New(
		notification.WithChannel("skip", channel),
		notification.WithChannel("fail", channel),
		notification.WithChannel("send", channel),
	)
	require.NoError(t, err)
	n := guardedNotice{
		notice: newNotice("skip", "fail", "send"),
		guard: func(_ context.Context, _ notification.Recipient, channel string) (bool, error) {
			switch channel {
			case "skip":
				return false, nil
			case "fail":
				return false, guardErr
			default:
				return true, nil
			}
		},
	}
	result, err := manager.Send(t.Context(), recipient{}, n)
	require.ErrorIs(t, err, guardErr)
	require.Len(t, result.Deliveries, 3)
	assert.Equal(t, notification.StatusSkipped, result.Deliveries[0].Status)
	assert.Nil(t, result.Deliveries[0].Err)
	assert.Equal(t, notification.StatusFailed, result.Deliveries[1].Status)
	assert.Equal(t, notification.StatusSent, result.Deliveries[2].Status)
	assert.Equal(t, []string{"send"}, sent)
}

func TestSendCancellationPreservesSuccessfulDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var sent []string
	channel := notification.ChannelFunc(func(_ context.Context, d notification.Delivery) error {
		sent = append(sent, d.Channel)
		cancel()
		return nil
	})
	manager, err := notification.New(
		notification.WithChannel("first", channel),
		notification.WithChannel("second", channel),
		notification.WithChannel("third", channel),
	)
	require.NoError(t, err)
	result, err := manager.Send(ctx, recipient{}, newNotice("first", "second", "third"))
	require.ErrorIs(t, err, context.Canceled)
	require.Len(t, result.Deliveries, 3)
	assert.Equal(t, notification.StatusSent, result.Deliveries[0].Status)
	assert.Equal(t, notification.StatusNotAttempted, result.Deliveries[1].Status)
	assert.Equal(t, notification.StatusNotAttempted, result.Deliveries[2].Status)
	assert.Equal(t, []string{"first"}, sent)

	result, err = manager.Send(ctx, recipient{}, newNotice("first"))
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, result)
}

func TestSendCancellationDuringSelectionAndGuard(t *testing.T) {
	for _, where := range []string{"via", "guard"} {
		t.Run(where, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			manager, err := notification.New(notification.WithChannel("mail", notification.ChannelFunc(
				func(context.Context, notification.Delivery) error { calls++; return nil },
			)))
			require.NoError(t, err)
			n := guardedNotice{
				notice: notice{kind: "test", via: func(context.Context, notification.Recipient) ([]string, error) {
					if where == "via" {
						cancel()
					}
					return []string{"mail"}, nil
				}},
				guard: func(context.Context, notification.Recipient, string) (bool, error) {
					cancel()
					return true, nil
				},
			}
			result, err := manager.Send(ctx, recipient{}, n)
			require.ErrorIs(t, err, context.Canceled)
			assert.Zero(t, calls)
			if where == "guard" {
				require.Len(t, result.Deliveries, 1)
				assert.Equal(t, notification.StatusNotAttempted, result.Deliveries[0].Status)
			} else {
				assert.Zero(t, result)
			}
		})
	}
}

func TestDeliveryIDsAreStableAndScoped(t *testing.T) {
	calls := 0
	manager, err := notification.New(notification.WithChannel("mail", notification.ChannelFunc(
		func(context.Context, notification.Delivery) error { calls++; return nil },
	)))
	require.NoError(t, err)
	key := notification.RecipientKey{TenantID: "acme", Type: "user", ID: "42"}
	send := func(key notification.RecipientKey, options ...notification.SendOption) notification.Result {
		result, err := manager.Send(t.Context(), recipient{key: key}, newNotice("mail"), options...)
		require.NoError(t, err)
		return result
	}
	first := send(key, notification.WithID("invoice:123"))
	assert.Equal(t, "449b9336bb3c302f4b0b0804e1e2712bb1be1791e1c5a875c041620d2b0d9364", first.Deliveries[0].ID)
	second := send(key, notification.WithID("invoice:123"))
	assert.Equal(t, first.Deliveries[0].ID, second.Deliveries[0].ID)
	assert.Equal(t, 2, calls, "stable IDs do not deduplicate sends")
	for _, other := range []notification.RecipientKey{
		{TenantID: "other", Type: "user", ID: "42"},
		{TenantID: "acme", Type: "admin", ID: "42"},
		{TenantID: "acme", Type: "user", ID: "43"},
	} {
		assert.NotEqual(t, first.Deliveries[0].ID, send(other, notification.WithID("invoice:123")).Deliveries[0].ID)
	}
	assert.NotEqual(t, send(key).NotificationID, send(key).NotificationID)
	// These keys would collide if identity parts were simply joined with ':'.
	one := send(notification.RecipientKey{Type: "a:b", ID: "c"}, notification.WithID("same"))
	two := send(notification.RecipientKey{Type: "a", ID: "b:c"}, notification.WithID("same"))
	assert.NotEqual(t, one.Deliveries[0].ID, two.Deliveries[0].ID)
}

func TestSendNoChannelsAndZeroManager(t *testing.T) {
	var manager notification.Manager
	result, err := manager.Send(t.Context(), recipient{}, newNotice())
	require.NoError(t, err)
	assert.NotEmpty(t, result.NotificationID)
	assert.Empty(t, result.Deliveries)
	_, err = manager.Send(t.Context(), recipient{}, newNotice("missing"))
	require.ErrorIs(t, err, notification.ErrUnknownChannel)
}

func TestSendJoinsIndependentChannelFailures(t *testing.T) {
	firstErr, secondErr := errors.New("first failed"), errors.New("second failed")
	manager, err := notification.New(
		notification.WithChannel("first", notification.ChannelFunc(func(context.Context, notification.Delivery) error {
			return firstErr
		})),
		notification.WithChannel("second", notification.ChannelFunc(func(context.Context, notification.Delivery) error {
			return secondErr
		})),
	)
	require.NoError(t, err)
	result, err := manager.Send(t.Context(), recipient{}, newNotice("first", "second"))
	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	require.Len(t, result.Deliveries, 2)
	assert.Equal(t, notification.StatusFailed, result.Deliveries[0].Status)
	assert.Equal(t, notification.StatusFailed, result.Deliveries[1].Status)
}

func TestSendConcurrentSafe(t *testing.T) {
	var calls atomic.Int64
	manager, err := notification.New(notification.WithChannel("mail", notification.ChannelFunc(
		func(context.Context, notification.Delivery) error { calls.Add(1); return nil },
	)))
	require.NoError(t, err)
	const count = 64
	results := make(chan notification.Result, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Go(func() {
			result, err := manager.Send(t.Context(), recipient{}, newNotice("mail"))
			errs <- err
			results <- result
		})
	}
	wg.Wait()
	close(errs)
	close(results)
	for err := range errs {
		require.NoError(t, err)
	}
	ids := make(map[string]bool)
	for result := range results {
		assert.False(t, ids[result.NotificationID])
		ids[result.NotificationID] = true
	}
	assert.Equal(t, int64(count), calls.Load())
}

func TestSendPanicsOnNilManagerOrContext(t *testing.T) {
	var manager *notification.Manager
	assert.Panics(t, func() { _, _ = manager.Send(t.Context(), recipient{}, newNotice()) })
	manager, err := notification.New()
	require.NoError(t, err)
	assert.Panics(t, func() { _, _ = manager.Send(nil, recipient{}, newNotice()) }) //nolint:staticcheck // Verify the documented nil-context panic.
}

func TestRecipientKeyAndVersion(t *testing.T) {
	assert.True(t, (notification.RecipientKey{Type: "user", ID: "1"}).Valid())
	assert.False(t, (notification.RecipientKey{}).Valid())
	assert.False(t, (notification.RecipientKey{Type: "user", ID: " "}).Valid())
	assert.False(t, (notification.RecipientKey{Type: "\t", ID: "1"}).Valid())
	assert.False(t, (notification.RecipientKey{TenantID: "\xff", Type: "user", ID: "1"}).Valid())
	assert.False(t, (notification.RecipientKey{Type: "user", ID: "\xff"}).Valid())
	assert.False(t, (notification.RecipientKey{Type: "\xff", ID: "1"}).Valid())
	assert.Regexp(t, `^\d+\.\d+\.\d+$`, notification.Version())
}
