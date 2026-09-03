package jsonrpc

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: "null"},
		{name: "string", value: "request-id", want: "request-id"},
		{name: "int", value: int(1), want: "1"},
		{name: "int8", value: int8(2), want: "2"},
		{name: "int16", value: int16(3), want: "3"},
		{name: "int32", value: int32(4), want: "4"},
		{name: "int64", value: int64(5), want: "5"},
		{name: "uint", value: uint(6), want: "6"},
		{name: "uint8", value: uint8(7), want: "7"},
		{name: "uint16", value: uint16(8), want: "8"},
		{name: "uint32", value: uint32(9), want: "9"},
		{name: "uint64", value: uint64(10), want: "10"},
		{name: "uintptr", value: uintptr(11), want: "11"},
		{name: "float32", value: float32(12.5), want: "12.5"},
		{name: "float64", value: 13.5, want: "13.5"},
		{name: "complex64", value: complex64(complex(14.5, 1)), want: "14.5"},
		{name: "complex128", value: complex(15.5, 1), want: "15.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := NewID(tt.value)
			require.NotNil(t, id)
			assert.Equal(t, tt.want, id.String())
		})
	}

	assert.Nil(t, NewID(true))
}

func TestIDJSON(t *testing.T) {
	t.Run("marshal", func(t *testing.T) {
		tests := []struct {
			name string
			id   ID
			want string
		}{
			{name: "nil", id: *NewID(nil), want: "null"},
			{name: "string", id: *NewID("request-id"), want: `"request-id"`},
			{name: "number", id: *NewID(12.5), want: "12.5"},
			{name: "zero value", id: ID{}, want: "null"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.id.MarshalJSON()
				require.NoError(t, err)
				assert.JSONEq(t, tt.want, string(got))
			})
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		tests := []struct {
			name string
			data string
			want string
		}{
			{name: "nil", data: "null", want: "null"},
			{name: "string", data: `"request-id"`, want: "request-id"},
			{name: "number", data: "12.5", want: "12.5"},
			{name: "unsupported value", data: "true", want: "null"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var id ID
				require.NoError(t, id.UnmarshalJSON([]byte(tt.data)))
				assert.Equal(t, tt.want, id.String())
			})
		}
	})
}

func TestUUIDGenerator(t *testing.T) {
	generator := NewUUIDGenerator()
	id := generator.Generate()

	require.NotNil(t, id)
	_, err := uuid.Parse(id.String())
	require.NoError(t, err)
}
