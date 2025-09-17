package retry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetry(t *testing.T) {
	customError := errors.New("custom error")
	retry := New(
		Attempts(3),
		Backoff(LinearBackoff(1)),
		Allow(OrAllowFuncs(DefaultAllow, func(err error) bool {
			return errors.Is(err, assert.AnError)
		})),
	)

	tests := []struct {
		name    string
		handler func(ctx context.Context, service, method string, request any) (response any, err error)
		want    func(t *testing.T, err error)
	}{
		{
			name: "test",
			handler: func(context.Context, string, string, any) (any, error) {
				return nil, assert.AnError
			},
			want: func(t *testing.T, err error) {
				assert.True(t, IsError(err))
			},
		},
		{
			name: "test",
			handler: func(context.Context, string, string, any) (any, error) {
				return nil, customError
			},
			want: func(t *testing.T, err error) {
				assert.Equal(t, customError, err)
			},
		},
		{
			name: "test",
			handler: func(context.Context, string, string, any) (any, error) {
				return nil, nil
			},
			want: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := retry(tt.handler)(t.Context(), "service", "test", nil)
			tt.want(t, err)
		})
	}
}
