package cache

import (
	"sync"
	"time"
)

type Snapshot[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func (s *Snapshot[K, V]) Lookup(key K, fn func() V) V {
	s.mu.RLock()
	v, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.data == nil {
			s.data = make(map[K]V)
		}

		if v, ok = s.data[key]; !ok {
			v = fn()
			s.data[key] = v
		}
	}
	return v
}

type SnapshotWithErr[K comparable, V any] Snapshot[K, valueWithError[V]]

type valueWithError[V any] struct {
	value V
	err   error
}

func (c *SnapshotWithErr[K, V]) Lookup(key K, fn func() (V, error)) (V, error) {
	c.mu.RLock()
	v, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.data == nil {
			c.data = make(map[K]valueWithError[V])
		}

		if v, ok = c.data[key]; !ok {
			v.value, v.err = fn()
			c.data[key] = v
		}
	}
	return v.value, v.err
}

type SnapshotWithExpireAndErr[K comparable, V any] Snapshot[K, valueWithExpireAndError[V]]

type valueWithExpireAndError[V any] struct {
	value   V
	expired time.Time
	err     error
}

func (c *SnapshotWithExpireAndErr[K, V]) Lookup(key K, fn func() (V, error), expire time.Duration) (V, error) {
	c.mu.RLock()
	v, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(v.expired) {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.data == nil {
			c.data = make(map[K]valueWithExpireAndError[V])
		}

		if v, ok = c.data[key]; !ok || time.Now().After(v.expired) {
			v.value, v.err = fn()
			v.expired = time.Now().Add(expire)
			c.data[key] = v
		}
	}
	return v.value, v.err
}
