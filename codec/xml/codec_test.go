package xml //nolint:revive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var xmlBytes = []byte(`<TestMessage><name>test</name><value>123</value></TestMessage>`)

type TestMessage struct {
	Name  string `xml:"name"`
	Value int32  `xml:"value"`
}

func TestCodec(t *testing.T) {
	msg := &TestMessage{
		Name:  "test",
		Value: 123,
	}

	data, err := Codec.Marshal(msg)
	require.NoError(t, err)
	assert.Equal(t, xmlBytes, data)

	newMsg := &TestMessage{}
	err = Codec.Unmarshal(data, newMsg)
	require.NoError(t, err)

	assert.Equal(t, msg.Name, newMsg.Name)
	assert.Equal(t, msg.Value, newMsg.Value)

	// Test Unmarshal with XML bytes
	err = Codec.Unmarshal(xmlBytes, newMsg)
	require.NoError(t, err)
	assert.Equal(t, msg.Name, newMsg.Name)
	assert.Equal(t, msg.Value, newMsg.Value)
}
