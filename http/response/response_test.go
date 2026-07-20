package response_test

import (
	"encoding/json"
	"testing"

	"github.com/go-fries/fries/http/response/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccess(t *testing.T) {
	t.Parallel()

	data := map[string]int{"id": 11}
	body := response.Success("working properly", data, response.WithCode(10000))

	assert.True(t, body.Status)
	require.NotNil(t, body.Code)
	assert.Equal(t, 10000, *body.Code)
	assert.Equal(t, "working properly", body.Message)
	assert.Equal(t, data, body.Data)
}

func TestFailure(t *testing.T) {
	t.Parallel()

	data := map[string][]string{"name": {"name is required"}}
	body := response.Failure(
		"invalid request",
		response.WithCode(10422),
		response.WithData(data),
	)

	assert.False(t, body.Status)
	require.NotNil(t, body.Code)
	assert.Equal(t, 10422, *body.Code)
	assert.Equal(t, "invalid request", body.Message)
	assert.Equal(t, data, body.Data)
}

func TestBodyJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body response.Body
		want string
	}{
		{
			name: "omits unset code and nil data",
			body: response.Success("ok", nil),
			want: `{"status":true,"message":"ok"}`,
		},
		{
			name: "keeps zero code and omits nil data",
			body: response.Success("ok", nil, response.WithCode(0)),
			want: `{"status":true,"code":0,"message":"ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload, err := json.Marshal(tt.body)

			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(payload))
		})
	}
}
