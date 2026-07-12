package xml_test

import (
	"testing"

	"github.com/go-fries/fries/codec/v4"
	"github.com/go-fries/fries/codec/xml/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type message struct {
	Name  string `xml:"name"`
	Value int    `xml:"value"`
}

func TestCodec(t *testing.T) {
	c := xml.Codec{}
	assert.Implements(t, (*codec.Codec)(nil), c)

	want := message{Name: "test", Value: 123}
	data, err := c.Marshal(want)
	require.NoError(t, err)

	var got message
	require.NoError(t, c.Unmarshal(data, &got))
	assert.Equal(t, want, got)
}

func TestCodecErrors(t *testing.T) {
	c := xml.Codec{}

	_, err := c.Marshal(make(chan int))
	assert.Error(t, err)
	assert.Error(t, c.Unmarshal([]byte("<message>"), &message{}))
}
