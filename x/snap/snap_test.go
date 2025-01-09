package snap

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSnapshot_Basic(t *testing.T) {
	var (
		now   = time.Now()
		value int
	)
	snap := New(func() (int, error) {
		return rand.Intn(1000), nil
	}, Interval[int](time.Millisecond*500), Async[int](false))

	for {
		oldValue := value

		value = snap.Get()
		assert.True(t, value >= 0 && value < 1000)

		if oldValue != value {
			break
		}
	}
	assert.True(t, time.Since(now) < time.Millisecond*1000*2)
}

type User struct {
	Name string
	Age  int
}

func TestSnapshot_Struct(t *testing.T) {
	var (
		internal = time.Millisecond * 500
		now      = time.Now()
		age      int
	)
	snap := New(func() (*User, error) {
		return &User{
			Name: "test",
			Age:  rand.Intn(100),
		}, nil
	}, Interval[*User](internal), Async[*User](false))

	for {
		oldAge := age
		assert.Equal(t, "test", snap.Get().Name)

		age = snap.Get().Age
		assert.True(t, age >= 0 && age < 100)
		if oldAge != age {
			break
		}
	}
	assert.True(t, time.Since(now) < internal*2)
}
