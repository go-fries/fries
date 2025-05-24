package sonic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSonic(t *testing.T) {
	c1, c2 := Codec, Codec

	assert.Same(t, c1, c2)

	data := map[string]any{
		"foo": "bar",
	}

	// marshal
	bytes, err := c1.Marshal(data)
	assert.NoError(t, err)

	// unmarshal
	dest := make(map[string]any)
	assert.NoError(t, c1.Unmarshal(bytes, &dest))
}

func BenchmarkSonicCodec_Marshal(b *testing.B) {
	data := map[string]any{
		"foo": "bar",
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Codec.Marshal(data)
			assert.NoError(b, err)
		}
	})
}

func BenchmarkSonicCodec_Unmarshal(b *testing.B) {
	data := map[string]any{
		"foo": "bar",
	}

	bytes, err := Codec.Marshal(data)
	assert.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var dest map[string]any
			assert.NoError(b, Codec.Unmarshal(bytes, &dest))
		}
	})
}
