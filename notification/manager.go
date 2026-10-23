package notification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-fries/fries/notification/v4/internal/nilvalue"
)

// Manager delivers notifications using a fixed set of channels. It is safe for
// concurrent Send calls when its channels and application dependencies are.
// The zero value has no registered channels. Construct a Manager with New.
type Manager struct {
	config config
}

// New validates channel registrations and constructs a Manager.
func New(options ...Option) (*Manager, error) {
	c, err := newConfig(options...)
	if err != nil {
		return nil, err
	}
	return &Manager{config: c}, nil
}

// Send synchronously sends n to one recipient. It preflights the notification,
// options and selected channel registrations before attempting any delivery.
// Duplicate channel names are deduplicated in first-seen order.
//
// Ordinary guard/channel errors do not stop other channels; their DeliveryErrors
// are joined in the returned error. Cancellation prevents unstarted channels
// from running. A channel that returns nil remains sent even if ctx was canceled
// during its call. Send never retries and does not recover application panics.
// It panics if the Manager or ctx is nil.
func (m *Manager) Send(ctx context.Context, to Recipient, n Notification, options ...SendOption) (Result, error) {
	if m == nil || ctx == nil {
		panic("notification: nil manager or context")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if nilvalue.IsNil(to) {
		return Result{}, ErrInvalidRecipient
	}
	key := to.NotificationKey()
	if key != (RecipientKey{}) && !key.Valid() {
		return Result{}, ErrInvalidRecipient
	}
	if nilvalue.IsNil(n) || strings.TrimSpace(n.Type()) == "" {
		return Result{}, ErrInvalidNotification
	}
	c, err := newSendConfig(options...)
	if err != nil {
		return Result{}, err
	}
	channels, err := m.selectChannels(ctx, to, n)
	if err != nil {
		return Result{}, err
	}
	if c.id == "" {
		c.id = rand.Text()
	}
	result := Result{
		NotificationID: c.id,
		Deliveries:     make([]DeliveryResult, 0, len(channels)),
	}
	createdAt := time.Now().UTC()
	errs := make([]error, 0, len(channels))
	for _, name := range channels {
		delivery := Delivery{
			ID:             deliveryID(c.id, key, name),
			NotificationID: c.id,
			Channel:        name,
			Recipient:      to,
			Notification:   n,
			CreatedAt:      createdAt,
		}
		outcome := m.deliver(ctx, delivery)
		result.Deliveries = append(result.Deliveries, outcome)
		if outcome.Err != nil {
			errs = append(errs, outcome.Err)
		}
	}
	return result, errors.Join(errs...)
}

func (m *Manager) selectChannels(ctx context.Context, to Recipient, n Notification) ([]string, error) {
	names, err := n.Via(ctx, to)
	if err != nil {
		return nil, fmt.Errorf("notification: select channels: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(names))
	channels := make([]string, 0, len(names))
	for _, name := range names {
		if _, exists := m.config.channels[name]; !exists {
			return nil, fmt.Errorf("%w: %q", ErrUnknownChannel, name)
		}
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			channels = append(channels, name)
		}
	}
	return channels, nil
}

func (m *Manager) deliver(ctx context.Context, d Delivery) DeliveryResult {
	result := DeliveryResult{ID: d.ID, Channel: d.Channel}
	if err := ctx.Err(); err != nil {
		return failedResult(result, StatusNotAttempted, err)
	}
	if guard, ok := d.Notification.(SendGuard); ok {
		allowed, err := guard.ShouldSend(ctx, d.Recipient, d.Channel)
		if err != nil {
			return failedResult(result, StatusFailed, err)
		}
		if !allowed {
			result.Status = StatusSkipped
			return result
		}
	}
	if err := ctx.Err(); err != nil {
		return failedResult(result, StatusNotAttempted, err)
	}
	if err := m.config.channels[d.Channel].Send(ctx, d); err != nil {
		return failedResult(result, StatusFailed, err)
	}
	result.Status = StatusSent
	return result
}

func failedResult(result DeliveryResult, status Status, err error) DeliveryResult {
	result.Status = status
	result.Err = &DeliveryError{ID: result.ID, Channel: result.Channel, Err: err}
	return result
}

func deliveryID(id string, key RecipientKey, channel string) string {
	// Length-prefixed parts avoid delimiter collisions and preserve arbitrary bytes.
	data := make([]byte, 0, 128)
	for _, part := range [5]string{id, key.TenantID, key.Type, key.ID, channel} {
		data = binary.AppendUvarint(data, uint64(len(part)))
		data = append(data, part...)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
