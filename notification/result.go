package notification

// Status is the outcome of one channel delivery.
type Status string

const (
	// StatusSent means the channel returned nil.
	StatusSent Status = "sent"
	// StatusSkipped means a SendGuard returned false with no error.
	StatusSkipped Status = "skipped"
	// StatusFailed means a guard or channel returned an error.
	StatusFailed Status = "failed"
	// StatusNotAttempted means cancellation prevented a channel from starting.
	StatusNotAttempted Status = "not_attempted"
)

// Result reports a logical notification to one recipient. Deliveries remain in
// Via order. Preflight failures return a zero Result and have no side effects.
type Result struct {
	NotificationID string
	Deliveries     []DeliveryResult
}

// DeliveryResult reports the outcome and optional error of one channel.
// Err is a *DeliveryError for failed and not-attempted deliveries, otherwise nil.
type DeliveryResult struct {
	ID      string
	Channel string
	Status  Status
	Err     error
}
