package nilvalue_test

import (
	"testing"

	"github.com/go-fries/fries/notification/v4/internal/nilvalue"
	"github.com/stretchr/testify/assert"
)

func TestIsNil(t *testing.T) {
	var pointer *int
	var function func()
	var channel chan int
	var values []int
	var mapping map[string]int
	for _, value := range []any{nil, pointer, function, channel, values, mapping} {
		assert.True(t, nilvalue.IsNil(value))
	}
	for _, value := range []any{0, "", struct{}{}, new(int), func() {}, []int{}, map[string]int{}} {
		assert.False(t, nilvalue.IsNil(value))
	}
}
