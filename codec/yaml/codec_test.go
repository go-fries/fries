package yaml_test

import (
	"testing"

	"github.com/go-fries/fries/codec/v4"
	"github.com/go-fries/fries/codec/yaml/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type message struct {
	Name  string `yaml:"name"`
	Value int    `yaml:"value"`
}

func TestCodec(t *testing.T) {
	c := yaml.Codec{}
	assert.Implements(t, (*codec.Codec)(nil), c)

	want := message{Name: "test", Value: 123}
	data, err := c.Marshal(want)
	require.NoError(t, err)

	var got message
	require.NoError(t, c.Unmarshal(data, &got))
	assert.Equal(t, want, got)
}

func TestCodecErrors(t *testing.T) {
	c := yaml.Codec{}

	assert.Error(t, c.Unmarshal([]byte("value: ["), &message{}))
}
