package coordinator

import "sync"

type Coordinator struct {
	c    chan struct{}
	once sync.Once
}

func NewCoordinator() *Coordinator {
	return &Coordinator{
		c: make(chan struct{}),
	}
}

func (c *Coordinator) Done() <-chan struct{} {
	return c.c
}

func (c *Coordinator) Close() {
	c.once.Do(func() {
		close(c.c)
	})
}
