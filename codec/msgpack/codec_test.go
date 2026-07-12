package msgpack_test

import (
	"testing"

	"github.com/go-fries/fries/codec/msgpack/v4"
	"github.com/go-fries/fries/codec/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type message struct {
	Name  string `msgpack:"name"`
	Value int    `msgpack:"value"`
}

func TestCodec(t *testing.T) {
	c := msgpack.Codec{}
	assert.Implements(t, (*codec.Codec)(nil), c)

	want := message{Name: "test", Value: 123}
	data, err := c.Marshal(want)
	require.NoError(t, err)

	var got message
	require.NoError(t, c.Unmarshal(data, &got))
	assert.Equal(t, want, got)
}

func TestCodecErrors(t *testing.T) {
	c := msgpack.Codec{}

	_, err := c.Marshal(make(chan int))
	assert.Error(t, err)
	assert.Error(t, c.Unmarshal([]byte{0xc1}, &message{}))
}

func BenchmarkCodecMarshal(b *testing.B) {
	c := msgpack.Codec{}
	value := message{Name: "test", Value: 123}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := c.Marshal(value); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCodecUnmarshal(b *testing.B) {
	c := msgpack.Codec{}
	data, err := c.Marshal(message{Name: "test", Value: 123})
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		var value message
		if err := c.Unmarshal(data, &value); err != nil {
			b.Fatal(err)
		}
	}
}
