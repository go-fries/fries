package coroutines

import (
	"testing"
	"time"

	"github.com/go-fries/fries/support/v3"
	"github.com/stretchr/testify/assert"
)

func TestWorker(t *testing.T) {
	var (
		fns []func()
		ch  = make(chan struct{}, 200)
		now = time.Now()
	)
	_ = support.Repeat(func() error {
		ch <- struct{}{}
		return nil
	}, 200)
	_ = support.Repeat(func() error {
		fns = append(fns, func() {
			time.Sleep(100 * time.Millisecond)
			<-ch
		})
		return nil
	}, 100)

	assert.Len(t, ch, 200)
	assert.Len(t, fns, 100)
	assert.True(t, time.Since(now) < 100*time.Millisecond)

	w := NewWorker(10)
	defer w.Close()

	// the first time
	w.Push(fns...)
	w.Wait()

	assert.True(t, time.Since(now) < 2*time.Second)
	assert.True(t, time.Since(now) > 1*time.Second)
	assert.Len(t, ch, 100)

	// the second time
	now2 := time.Now()
	w.Push(fns...)
	w.Wait()
	assert.True(t, time.Since(now2) < 2*time.Second)
	assert.True(t, time.Since(now2) > 1*time.Second)
	assert.Len(t, ch, 0)
}
