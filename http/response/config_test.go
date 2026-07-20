package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()

	defaultData := map[string]int{"id": 11}
	replacementData := map[string]string{"field": "invalid"}

	tests := []struct {
		name     string
		data     any
		options  []Option
		wantCode *int
		wantData any
	}{
		{
			name:     "keeps default data",
			data:     defaultData,
			wantData: defaultData,
		},
		{
			name:     "sets code",
			options:  []Option{WithCode(200)},
			wantCode: intPointer(200),
		},
		{
			name:     "keeps zero code",
			options:  []Option{WithCode(0)},
			wantCode: intPointer(0),
		},
		{
			name:     "replaces data",
			data:     defaultData,
			options:  []Option{WithData(replacementData)},
			wantData: replacementData,
		},
		{
			name:     "ignores nil option",
			data:     defaultData,
			options:  []Option{nil},
			wantData: defaultData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newConfig(tt.data, tt.options...)

			if tt.wantCode == nil {
				assert.Nil(t, c.code)
			} else {
				require.NotNil(t, c.code)
				assert.Equal(t, *tt.wantCode, *c.code)
			}
			assert.Equal(t, tt.wantData, c.data)
		})
	}
}

func TestWithCodeDoesNotSharePointers(t *testing.T) {
	t.Parallel()

	option := WithCode(200)
	first := newConfig(nil, option)
	second := newConfig(nil, option)
	require.NotNil(t, first.code)
	require.NotNil(t, second.code)

	*first.code = 500

	assert.Equal(t, 500, *first.code)
	assert.Equal(t, 200, *second.code)
}

func intPointer(value int) *int {
	return &value
}
