package queue

import "maps"

// QueueOption is an option that applies to producers, task enqueueing, and worker consumption.
type QueueOption interface {
	ProducerOption
	EnqueueOption
	WorkerOption
}

// ObserverOption is an option that applies to both producers and workers.
type ObserverOption interface {
	ProducerOption
	WorkerOption
}

type queueOption struct {
	name string
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

func (o queueOption) applyProducer(c *producerConfig) {
	if o.name != "" {
		c.queue = o.name
	}
}

func (o queueOption) applyWorker(c *workerConfig) {
	if o.name != "" {
		c.queue = o.name
	}
}

type observerOption struct {
	observer Observer
}

// WithObserver returns an option that sets the observer used by producers or workers.
func WithObserver(observer Observer) ObserverOption {
	return observerOption{observer: observer}
}

func (o observerOption) applyProducer(c *producerConfig) {
	c.observer = o.observer
}

func (o observerOption) applyWorker(c *workerConfig) {
	c.observer = o.observer
}

type metadataOption struct {
	metadata map[string]string
}

// WithMetadata adds default or task-specific metadata values.
func WithMetadata(metadata map[string]string) metadataOption {
	return metadataOption{metadata: metadata}
}

func (o metadataOption) applyProducer(c *producerConfig) {
	if len(o.metadata) == 0 {
		return
	}
	if c.metadata == nil {
		c.metadata = make(map[string]string, len(o.metadata))
	}
	maps.Copy(c.metadata, o.metadata)
}

func (o metadataOption) applyEnqueue(c *enqueueConfig) {
	if len(o.metadata) == 0 {
		return
	}
	if c.metadata == nil {
		c.metadata = make(map[string]string, len(o.metadata))
	}
	maps.Copy(c.metadata, o.metadata)
}
