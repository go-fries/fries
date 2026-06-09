package queue

type queueOption struct {
	name string
}

// QueueOption is an option that applies to both task enqueueing and worker consumption.
type QueueOption interface {
	EnqueueOption
	WorkerOption
}

// WithQueue returns an option that selects the queue used to enqueue or consume tasks.
func WithQueue(name string) QueueOption {
	return queueOption{name: name}
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
