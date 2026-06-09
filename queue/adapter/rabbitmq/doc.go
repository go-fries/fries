// Package rabbitmq adapts RabbitMQ AMQP 0.9.1 queues to the queue component.
//
// The adapter uses durable queues and publisher confirms by default. Publish
// operations use the caller's context as the publish and confirmation deadline.
//
// Delayed retries are implemented with per-delay TTL queues that dead-letter
// back to the ready queue. Failed deliveries are moved to an adapter-managed
// dead-letter queue. The adapter does not perform automatic connection
// recovery; applications should recreate closed AMQP connections outside this
// package.
package rabbitmq
