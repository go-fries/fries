package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var yamlBytes = []byte(`name: test
value: 123`)

type TestMessage struct {
	Name  string `yaml:"name"`
	Value int32  `yaml:"value"`
}

func TestCodec(t *testing.T) {
	msg := &TestMessage{
		Name:  "test",
		Value: 123,
	}

	data, err := Codec.Marshal(msg)
	require.NoError(t, err)
	assert.Equal(t, yamlBytes, data)

	newMsg := &TestMessage{}
	err = Codec.Unmarshal(data, newMsg)
	require.NoError(t, err)

	assert.Equal(t, msg.Name, newMsg.Name)
	assert.Equal(t, msg.Value, newMsg.Value)

	// Test Unmarshal with YAML bytes
	err = Codec.Unmarshal(yamlBytes, newMsg)
	require.NoError(t, err)
	assert.Equal(t, msg.Name, newMsg.Name)
	assert.Equal(t, msg.Value, newMsg.Value)
}
