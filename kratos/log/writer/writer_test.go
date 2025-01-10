package writer

import (
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/stretchr/testify/assert"
)

var result = make(chan data, 1)

type data struct {
	level   log.Level
	keyvals []any
}

type logger struct{}

func (l *logger) Log(level log.Level, keyvals ...any) error {
	result <- data{
		level:   level,
		keyvals: keyvals,
	}
	return nil
}

func TestWriter(t *testing.T) {
	w := New(&logger{})

	n, err := w.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	d := <-result
	assert.Equal(t, log.LevelInfo, d.level)
	assert.Equal(t, "msg", d.keyvals[0])
	assert.Equal(t, "test", d.keyvals[1])
}
