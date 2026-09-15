package timeout

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeoutHandlerResult(t *testing.T) {
	handlerErr := errors.New("handler failed")
	for _, tt := range []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "handler error", err: handlerErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				type call struct {
					ctx             context.Context
					service, method string
					request         any
				}
				calls := make(chan call, 1)
				handler := New()(func(ctx context.Context, service, method string, request any) (any, error) {
					calls <- call{ctx, service, method, request}
					return "response", tt.err
				})
				started := time.Now()
				response, err := handler(t.Context(), "service", "method", "request")
				assert.Equal(t, "response", response)
				assert.ErrorIs(t, err, tt.err)
				require.Len(t, calls, 1)
				got := <-calls
				assert.Equal(t, "service", got.service)
				assert.Equal(t, "method", got.method)
				assert.Equal(t, "request", got.request)
				deadline, ok := got.ctx.Deadline()
				require.True(t, ok)
				assert.Equal(t, started.Add(defaultTimeout), deadline)
				assert.ErrorIs(t, got.ctx.Err(), context.Canceled)
				assert.Zero(t, time.Since(started))
			})
		})
	}
}

func TestTimeoutReturnsBeforeHandlerCompletes(t *testing.T) {
	for _, tt := range []struct {
		name          string
		parentTimeout time.Duration
		cancel        bool
		elapsed       time.Duration
		wantErr       error
	}{
		{name: "configured deadline", elapsed: time.Second, wantErr: ErrTimeout},
		{name: "shorter parent deadline", parentTimeout: 500 * time.Millisecond, elapsed: 500 * time.Millisecond, wantErr: ErrTimeout},
		{name: "parent cancellation", cancel: true, wantErr: context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tt.parentTimeout > 0 {
					var cancelDeadline context.CancelFunc
					ctx, cancelDeadline = context.WithTimeout(ctx, tt.parentTimeout)
					defer cancelDeadline()
				}
				release := make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				defer func() {
					unblock()
					synctest.Wait()
				}()
				started := make(chan context.Context, 1)
				completed := make(chan struct{}, 1)
				handler := New(Timeout(time.Second))(func(ctx context.Context, _, _ string, _ any) (any, error) {
					started <- ctx
					<-release // Deliberately ignore cancellation to exercise early return.
					completed <- struct{}{}
					return "late response", errors.New("late error")
				})
				type result struct {
					response any
					err      error
				}
				results := make(chan result, 1)
				begin := time.Now()
				go func() {
					response, err := handler(ctx, "service", "method", "request")
					results <- result{response, err}
				}()
				synctest.Wait()
				require.Len(t, started, 1)
				handlerCtx := <-started
				require.Empty(t, results)
				if tt.cancel {
					cancel()
				} else {
					time.Sleep(tt.elapsed - time.Nanosecond)
					synctest.Wait()
					assert.NoError(t, handlerCtx.Err())
					require.Empty(t, results, "middleware returned before its deadline")
					time.Sleep(time.Nanosecond)
				}
				synctest.Wait()
				require.Len(t, results, 1, "middleware waited for the blocked handler")
				got := <-results
				assert.Nil(t, got.response)
				assert.ErrorIs(t, got.err, tt.wantErr)
				assert.Equal(t, tt.elapsed, time.Since(begin))
				require.Empty(t, completed)
				unblock()
				synctest.Wait()
				require.Len(t, completed, 1)
			})
		})
	}
}
