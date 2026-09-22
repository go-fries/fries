package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-fries/fries/codec/v4"
	"github.com/go-fries/fries/retry/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlePayloadWorkerDecisions(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		handlerErr error
		invalid    bool
		settlement string
		delay      time.Duration
		reason     string
	}{
		{name: "success", settlement: "ack"},
		{name: "ordinary error", handlerErr: assert.AnError, settlement: "retry", delay: time.Second},
		{name: "discard", handlerErr: fmt.Errorf("already sent: %w", ErrDiscard), settlement: "ack"},
		{name: "retry after", handlerErr: fmt.Errorf("limited: %w", RetryAfter(5*time.Second)), settlement: "retry", delay: 5 * time.Second},
		{name: "dead letter", handlerErr: fmt.Errorf("invalid user: %w", DeadLetter("unknown user")), settlement: "dead letter", reason: "unknown user"},
		{name: "invalid payload", invalid: true, settlement: "retry", delay: time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			var calls []string
			var observedErr error
			worker := NewWorker(newTestQueue(),
				HandlePayload("send_email", func(received context.Context, payload emailPayload) error {
					assert.Equal(t, ctx, received)
					assert.Equal(t, emailPayload{UserID: 42, Subject: "welcome"}, payload)
					calls = append(calls, "handler")
					return tt.handlerErr
				}),
				WithBackoff(retry.Fixed(time.Second)),
				WithMiddleware(func(next Handler) Handler {
					return HandlerFunc(func(ctx context.Context, task *Task) error {
						calls = append(calls, "before")
						err := next.Handle(ctx, task)
						observedErr = err
						calls = append(calls, "after")
						return err
					})
				}),
			)
			payload := []byte(`{"user_id":42,"subject":"welcome"}`)
			if tt.invalid {
				payload = []byte(`{`)
			}
			var delay time.Duration
			var reason string
			delivery := &recordingDelivery{
				task: &Task{Type: "send_email", Payload: payload, Attempt: 1},
				ack: func(context.Context) error {
					calls = append(calls, "ack")
					return nil
				},
				retry: func(_ context.Context, value time.Duration) error {
					calls = append(calls, "retry")
					delay = value
					return nil
				},
				deadLetter: func(_ context.Context, value string) error {
					calls = append(calls, "dead letter")
					reason = value
					return nil
				},
			}

			require.NoError(t, worker.process(ctx, delivery))
			if tt.invalid {
				var syntaxErr *json.SyntaxError
				assert.ErrorAs(t, observedErr, &syntaxErr)
				assert.Equal(t, []string{"before", "after", tt.settlement}, calls)
			} else {
				assert.ErrorIs(t, observedErr, tt.handlerErr)
				assert.Equal(t, []string{"before", "handler", "after", tt.settlement}, calls)
			}
			assert.Equal(t, tt.delay, delay)
			assert.Equal(t, tt.reason, reason)
		})
	}
}

func TestHandlePayloadWithCodec(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		codec   codec.Codec
		payload string
		wantErr error
	}{
		{name: "custom codec", codec: passthroughCodec{}, payload: "hello"},
		{name: "nil codec uses JSON", payload: `"hello"`},
		{name: "decode error", codec: failingCodec{unmarshalErr: assert.AnError}, wantErr: assert.AnError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var received string
			called := false
			config := newWorkerConfig(HandlePayloadWithCodec("message", tt.codec, func(_ context.Context, payload string) error {
				called = true
				received = payload
				return nil
			}))
			handler := config.handlers["message"]
			require.NotNil(t, handler)
			err := handler.Handle(t.Context(), &Task{Payload: []byte(tt.payload)})
			require.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantErr == nil, called)
			if tt.wantErr == nil {
				assert.Equal(t, "hello", received)
			}
		})
	}
}

func TestHandlePayloadIgnoresEmptyTypeAndNilHandler(t *testing.T) {
	t.Parallel()

	handler := func(context.Context, emailPayload) error { return nil }
	config := newWorkerConfig(
		HandlePayload("", handler),
		HandlePayload[emailPayload]("ignored", nil),
		HandlePayloadWithCodec("", nil, handler),
		HandlePayloadWithCodec[emailPayload]("also_ignored", nil, nil),
	)
	assert.Empty(t, config.handlers)
}
