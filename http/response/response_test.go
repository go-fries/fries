package response_test

import (
	"encoding/json"
	"errors"
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

func TestFromError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		options     []response.Option
		wantStatus  bool
		wantCode    *int
		wantMessage string
		wantData    any
	}{
		{
			name:       "returns success for nil error",
			wantStatus: true,
		},
		{
			name:        "returns failure for non-nil error",
			err:         errors.New("invalid request"),
			wantMessage: "invalid request",
		},
		{
			name:       "applies options to success",
			options:    []response.Option{response.WithCode(0), response.WithData("result")},
			wantStatus: true,
			wantCode:   intPointer(0),
			wantData:   "result",
		},
		{
			name:        "applies options to failure",
			err:         errors.New("invalid request"),
			options:     []response.Option{response.WithCode(10422), response.WithData("details")},
			wantCode:    intPointer(10422),
			wantMessage: "invalid request",
			wantData:    "details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := response.FromError(tt.err, tt.options...)

			assert.Equal(t, tt.wantStatus, body.Status)
			if tt.wantCode == nil {
				assert.Nil(t, body.Code)
			} else {
				require.NotNil(t, body.Code)
				assert.Equal(t, *tt.wantCode, *body.Code)
			}
			assert.Equal(t, tt.wantMessage, body.Message)
			assert.Equal(t, tt.wantData, body.Data)
		})
	}
}

func TestBodyJSON(t *testing.T) {
	t.Parallel()

	var typedNilPointer *struct{}
	var typedNilSlice []int
	var typedNilMap map[string]int

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
		{
			name: "keeps zero data",
			body: response.Success("ok", 0),
			want: `{"status":true,"message":"ok","data":0}`,
		},
		{
			name: "keeps empty string data",
			body: response.Success("ok", ""),
			want: `{"status":true,"message":"ok","data":""}`,
		},
		{
			name: "keeps empty slice data",
			body: response.Success("ok", []int{}),
			want: `{"status":true,"message":"ok","data":[]}`,
		},
		{
			name: "keeps empty map data",
			body: response.Success("ok", map[string]int{}),
			want: `{"status":true,"message":"ok","data":{}}`,
		},
		{
			name: "encodes typed nil pointer as null",
			body: response.Success("ok", typedNilPointer),
			want: `{"status":true,"message":"ok","data":null}`,
		},
		{
			name: "encodes typed nil slice as null",
			body: response.Success("ok", typedNilSlice),
			want: `{"status":true,"message":"ok","data":null}`,
		},
		{
			name: "encodes typed nil map as null",
			body: response.Success("ok", typedNilMap),
			want: `{"status":true,"message":"ok","data":null}`,
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

func intPointer(value int) *int {
	return &value
}
