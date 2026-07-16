package event

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userEvent struct {
	name string
}

type orderEvent struct {
	id string
}

type userHandler struct {
	called *atomic.Int64
}

func (h *userHandler) Handle(context.Context, userEvent) error {
	h.called.Add(1)
	return nil
}

func TestDispatcherDispatchesExactTypesInRegistrationOrder(t *testing.T) {
	dispatcher := New()
	var calls []string

	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(_ context.Context, value userEvent) error {
			calls = append(calls, "first:"+value.name)
			return nil
		})),
		HandlerFor[orderEvent](HandlerFunc[orderEvent](func(_ context.Context, value orderEvent) error {
			calls = append(calls, "order:"+value.id)
			return nil
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(_ context.Context, value userEvent) error {
			calls = append(calls, "second:"+value.name)
			return nil
		})),
	)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{name: "alice"}))
	assert.Equal(t, []string{"first:alice", "second:alice"}, calls)

	calls = nil
	require.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{id: "42"}))
	assert.Equal(t, []string{"order:42"}, calls)
}

func TestDispatcherSupportsStructHandlersAndTypedNilPointers(t *testing.T) {
	dispatcher := New()
	var called atomic.Int64
	dispatcher.Subscribe(HandlerFor[userEvent](&userHandler{called: &called}))

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, called.Load())

	var received *userEvent
	dispatcher.Subscribe(HandlerFor[*userEvent](HandlerFunc[*userEvent](func(_ context.Context, value *userEvent) error {
		received = value
		return nil
	})))
	var value *userEvent
	require.NoError(t, dispatcher.Dispatch(t.Context(), value))
	assert.Nil(t, received)
}

func TestDispatcherReturnsNilWithoutMatchingListeners(t *testing.T) {
	dispatcher := New()
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		return assert.AnError
	})))

	assert.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{}))
}

func TestHandlerForValidation(t *testing.T) {
	t.Run("interface event type", func(t *testing.T) {
		type message interface{ message() }
		assert.Panics(t, func() {
			HandlerFor[message](HandlerFunc[message](func(context.Context, message) error {
				return nil
			}))
		})
	})

	t.Run("nil handler interface", func(t *testing.T) {
		var handler Handler[userEvent]
		assert.Panics(t, func() { HandlerFor[userEvent](handler) })
	})

	t.Run("typed nil handler", func(t *testing.T) {
		var handler HandlerFunc[userEvent]
		assert.Panics(t, func() { HandlerFor[userEvent](handler) })
	})
}

func TestSubscribeValidatesBatchBeforeMutation(t *testing.T) {
	dispatcher := New()
	var calls atomic.Int64
	listener := HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		calls.Add(1)
		return nil
	}))

	assert.Panics(t, func() { dispatcher.Subscribe(listener, nil) })
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.Zero(t, calls.Load())
}

func TestSubscriptionUnsubscribe(t *testing.T) {
	dispatcher := New()
	var userCalls, orderCalls atomic.Int64
	subscription := dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			userCalls.Add(1)
			return nil
		})),
		HandlerFor[orderEvent](HandlerFunc[orderEvent](func(context.Context, orderEvent) error {
			orderCalls.Add(1)
			return nil
		})),
	)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	require.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{}))
	assert.True(t, subscription.Unsubscribe())
	assert.False(t, subscription.Unsubscribe())
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	require.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{}))
	assert.EqualValues(t, 1, userCalls.Load())
	assert.EqualValues(t, 1, orderCalls.Load())
}

func TestSubscriptionWithoutListenersIsInactive(t *testing.T) {
	subscription := New().Subscribe()
	require.NotNil(t, subscription)
	assert.False(t, subscription.Unsubscribe())

	var nilSubscription *Subscription
	assert.False(t, nilSubscription.Unsubscribe())
}

func TestRepeatedListenerCreatesIndependentRegistrations(t *testing.T) {
	dispatcher := New()
	var calls atomic.Int64
	listener := HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		calls.Add(1)
		return nil
	}))
	dispatcher.Subscribe(listener, listener)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 2, calls.Load())
}

func TestDispatcherValidation(t *testing.T) {
	dispatcher := New(nil)
	assert.ErrorIs(t, dispatcher.Dispatch(nil, userEvent{}), ErrInvalidContext) //nolint:staticcheck // Verifies the nil context contract.
	assert.ErrorIs(t, dispatcher.Dispatch(t.Context(), nil), ErrNilEvent)

	var nilDispatcher *Dispatcher
	assert.Panics(t, func() { nilDispatcher.Subscribe() })
	assert.Panics(t, func() { _ = nilDispatcher.Dispatch(t.Context(), userEvent{}) })
}

func TestDispatchReturnsContextCauseBeforeStarting(t *testing.T) {
	dispatcher := New()
	var calls atomic.Int64
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		calls.Add(1)
		return nil
	})))

	cause := errors.New("stop")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	assert.ErrorIs(t, dispatcher.Dispatch(ctx, userEvent{}), cause)
	assert.Zero(t, calls.Load())
}

func TestDispatchReturnsExpiredDeadline(t *testing.T) {
	dispatcher := New()
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()

	assert.ErrorIs(t, dispatcher.Dispatch(ctx, userEvent{}), context.DeadlineExceeded)
}

func TestDispatchFailFast(t *testing.T) {
	dispatcher := New()
	firstError := errors.New("first")
	var calls []int
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			calls = append(calls, 1)
			return firstError
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			calls = append(calls, 2)
			return nil
		})),
	)

	err := dispatcher.Dispatch(t.Context(), userEvent{})
	assert.Same(t, firstError, err)
	assert.Equal(t, []int{1}, calls)
}

func TestDispatchContinueOnErrorJoinsInRegistrationOrder(t *testing.T) {
	dispatcher := New()
	firstError := errors.New("first")
	secondError := errors.New("second")
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			return firstError
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			return nil
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			return secondError
		})),
	)

	err := dispatcher.Dispatch(t.Context(), userEvent{}, ContinueOnError())
	assert.ErrorIs(t, err, firstError)
	assert.ErrorIs(t, err, secondError)
	assert.Equal(t, "first\nsecond", err.Error())
}

func TestDispatchJoinsExternalContextCauseAfterHandlerError(t *testing.T) {
	dispatcher := New()
	handlerError := errors.New("handler")
	contextError := errors.New("context")
	ctx, cancel := context.WithCancelCause(t.Context())
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		cancel(contextError)
		return handlerError
	})))

	err := dispatcher.Dispatch(ctx, userEvent{})
	assert.ErrorIs(t, err, handlerError)
	assert.ErrorIs(t, err, contextError)
	assert.Equal(t, "handler\ncontext", err.Error())
}

func TestDispatchDoesNotDuplicateContextCause(t *testing.T) {
	dispatcher := New()
	cause := errors.New("context")
	ctx, cancel := context.WithCancelCause(t.Context())
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		cancel(cause)
		return fmt.Errorf("handler: %w", cause)
	})))

	err := dispatcher.Dispatch(ctx, userEvent{})
	assert.Equal(t, "handler: context", err.Error())
}

func TestDispatchWithConcurrencyIsBoundedAndWaits(t *testing.T) {
	dispatcher := New()
	var current, maximum, completed atomic.Int64
	listeners := make([]Listener, 0, 6)
	for range 6 {
		listeners = append(listeners, HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			active := current.Add(1)
			for {
				observed := maximum.Load()
				if active <= observed || maximum.CompareAndSwap(observed, active) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			current.Add(-1)
			completed.Add(1)
			return nil
		})))
	}
	dispatcher.Subscribe(listeners...)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}, WithConcurrency(2)))
	assert.EqualValues(t, 2, maximum.Load())
	assert.EqualValues(t, 6, completed.Load())
}

func TestDispatchConcurrentFailFastCancelsRunningHandlers(t *testing.T) {
	dispatcher := New()
	firstStarted := make(chan struct{})
	listenerError := errors.New("listener")
	var thirdCalls atomic.Int64
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(ctx context.Context, _ userEvent) error {
			close(firstStarted)
			<-ctx.Done()
			return ctx.Err()
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			<-firstStarted
			return listenerError
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			thirdCalls.Add(1)
			return nil
		})),
	)

	err := dispatcher.Dispatch(t.Context(), userEvent{}, WithConcurrency(2))
	assert.ErrorIs(t, err, listenerError)
	assert.Zero(t, thirdCalls.Load())
}

func TestDispatchConcurrentContinueOnErrorKeepsRegistrationOrder(t *testing.T) {
	dispatcher := New()
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			time.Sleep(30 * time.Millisecond)
			return errors.New("first")
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			return errors.New("second")
		})),
	)

	err := dispatcher.Dispatch(t.Context(), userEvent{}, WithConcurrency(2), ContinueOnError())
	assert.Equal(t, "first\nsecond", err.Error())
}

func TestWithConcurrencyIgnoresNonPositiveValues(t *testing.T) {
	dispatcher := New()
	var order []int
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			order = append(order, 1)
			return nil
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			order = append(order, 2)
			return nil
		})),
	)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}, WithConcurrency(0), nil))
	assert.Equal(t, []int{1, 2}, order)
}

func TestHandlerCanUnsubscribeItself(t *testing.T) {
	dispatcher := New()
	var subscription *Subscription
	var firstCalls, secondCalls atomic.Int64
	subscription = dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			firstCalls.Add(1)
			assert.True(t, subscription.Unsubscribe())
			return nil
		})),
		HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
			secondCalls.Add(1)
			return nil
		})),
	)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, firstCalls.Load())
	assert.EqualValues(t, 1, secondCalls.Load())
}

func TestHandlerCanDispatchNestedEvent(t *testing.T) {
	dispatcher := New()
	var orderCalls atomic.Int64
	dispatcher.Subscribe(
		HandlerFor[userEvent](HandlerFunc[userEvent](func(ctx context.Context, _ userEvent) error {
			return dispatcher.Dispatch(ctx, orderEvent{})
		})),
		HandlerFor[orderEvent](HandlerFunc[orderEvent](func(context.Context, orderEvent) error {
			orderCalls.Add(1)
			return nil
		})),
	)

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, orderCalls.Load())
}

func TestSubscriptionChangesAffectNextDispatch(t *testing.T) {
	dispatcher := New()
	var secondCalls atomic.Int64
	var once sync.Once
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		once.Do(func() {
			dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
				secondCalls.Add(1)
				return nil
			})))
		})
		return nil
	})))

	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.Zero(t, secondCalls.Load())
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.EqualValues(t, 1, secondCalls.Load())
}

func TestDispatcherConcurrentUse(t *testing.T) {
	dispatcher := New()
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			subscription := dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
				return nil
			})))
			_ = dispatcher.Dispatch(t.Context(), userEvent{}, WithConcurrency(2), ContinueOnError())
			subscription.Unsubscribe()
		})
	}
	workers.Wait()
}

func TestMiddlewareOrderAndMatching(t *testing.T) {
	var calls []string
	middleware := func(name string) Middleware {
		return func(next Next) Next {
			return func(ctx context.Context, value any) error {
				calls = append(calls, name+":before")
				err := next(ctx, value)
				calls = append(calls, name+":after")
				return err
			}
		}
	}
	dispatcher := New(WithMiddleware(middleware("outer"), nil, middleware("inner")))
	dispatcher.Subscribe(HandlerFor[userEvent](HandlerFunc[userEvent](func(context.Context, userEvent) error {
		calls = append(calls, "handler")
		return nil
	})))

	require.NoError(t, dispatcher.Dispatch(t.Context(), orderEvent{}))
	assert.Empty(t, calls)
	require.NoError(t, dispatcher.Dispatch(t.Context(), userEvent{}))
	assert.Equal(t, []string{
		"outer:before", "inner:before", "handler", "inner:after", "outer:after",
	}, calls)
}
