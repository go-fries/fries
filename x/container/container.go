package container

import (
	"errors"
	"sync"
)

var ErrKeyNotFound = errors.New("the key is not found")

type Container interface {
	Set(key, value any)
	Get(key any) (any, error)
}

type container struct {
	data  map[any]any
	mutex sync.RWMutex
}

func New() Container {
	return &container{
		data: make(map[any]any),
	}
}

func (c *container) Set(key, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = value
}

func (c *container) Get(key any) (any, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if v, ok := c.data[key]; ok {
		return v, nil
	}
	return nil, ErrKeyNotFound
}
