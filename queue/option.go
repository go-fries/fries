package queue

type queueOption struct {
	name string
}

type observerOption struct {
	observer Observer
}

// QueueOption is an option that applies to both task enqueueing and worker consumption.
type QueueOption interface {
	EnqueueOption
	WorkerOption
}

// ObserverOption is an option that applies to both producers and workers.
type ObserverOption interface {
	ProducerOption
	WorkerOption
}

// WithQueue returns an option that selects the queue used to enqueue or consume tasks.
func WithQueue(name string) QueueOption {
	return queueOption{name: name}
}

// WithObserver returns an option that sets the observer used by producers or workers.
func WithObserver(observer Observer) ObserverOption {
	return observerOption{observer: observer}
}

func (o queueOption) applyEnqueue(c *enqueueConfig) {
	if o.name != "" {
		c.queue = o.name
	}
}

func (o queueOption) applyWorker(c *workerConfig) {
	if o.name != "" {
		c.queue = o.name
	}
}

func (o observerOption) applyProducer(c *producerConfig) {
	c.observer = o.observer
}

func (o observerOption) applyWorker(c *workerConfig) {
	c.observer = o.observer
}
