package multidriver

import (
	"math/rand"
	"sync/atomic"

	"entgo.io/ent/dialect"
)

type Policy interface {
	Resolve([]dialect.Driver) dialect.Driver
}

type PolicyFunc func([]dialect.Driver) dialect.Driver

func (f PolicyFunc) Resolve(drivers []dialect.Driver) dialect.Driver {
	return f(drivers)
}

func RoundRobinPolicy() Policy {
	var i int
	return PolicyFunc(func(drivers []dialect.Driver) dialect.Driver {
		i = (i + 1) % len(drivers)
		return drivers[i]
	})
}

func StrictRoundRobinPolicy() Policy {
	var i atomic.Int64
	return PolicyFunc(func(drivers []dialect.Driver) dialect.Driver {
		return drivers[int(i.Add(1))%len(drivers)]
	})
}

func RandomPolicy() Policy {
	return PolicyFunc(func(drivers []dialect.Driver) dialect.Driver {
		return drivers[rand.Intn(len(drivers))]
	})
}
