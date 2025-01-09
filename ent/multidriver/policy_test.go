package multidriver

import (
	"sync"
	"sync/atomic"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/assert"
)

var (
	driver1 = &dialect.DebugDriver{}
	driver2 = &dialect.DebugDriver{}
	driver3 = &dialect.DebugDriver{}
)

// var driver1, driver2, driver3 dialect.Driver

func TestPolicy_RoundRobinPolicy(t *testing.T) {
	p1 := RoundRobinPolicy()
	p2 := StrictRoundRobinPolicy()
	drivers := []dialect.Driver{driver1, driver2, driver3}

	for i := 0; i < 10; i++ {
		assert.Same(t, drivers[(i+1)%len(drivers)], p1.Resolve(drivers))
		assert.Same(t, drivers[(i+1)%len(drivers)], p2.Resolve(drivers))
	}
}

func TestPolicy_RandomPolicy(t *testing.T) {
	p := RandomPolicy()
	drivers := []dialect.Driver{driver1, driver2, driver3}

	for i := 0; i < 10; i++ {
		assert.Contains(t, drivers, p.Resolve(drivers))
	}
}

func BenchmarkPolicy_StrictRoundRobinPolicy(b *testing.B) {
	p := StrictRoundRobinPolicy()
	drivers := []dialect.Driver{driver1, driver2, driver3}

	var i int64
	var mu sync.Mutex
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			assert.Same(b, drivers[int(atomic.AddInt64(&i, 1))%len(drivers)], p.Resolve(drivers))
			mu.Unlock()
		}
	})
}
