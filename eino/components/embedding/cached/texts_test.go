package cached

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextContext(t *testing.T) {
	for _, tt := range []struct {
		text string
	}{
		{"text1"},
		{"text2"},
		{"text3"},
	} {
		t.Run(tt.text, func(t *testing.T) {
			ctx := contextWithText(t.Context(), tt.text)

			retrievedText, ok := TextFromContext(ctx)
			assert.True(t, ok, "expected text to be present in context")
			assert.Equal(t, tt.text, retrievedText, "expected retrieved text to match original text")
		})
	}
}
