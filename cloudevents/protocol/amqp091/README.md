# AMQP 0.9.1 Protocol Binding for CloudEvents

## Overview

This component implements the AMQP 0.9.1 protocol binding for CloudEvents, following the [CloudEvents AMQP Protocol Binding Specification](https://github.com/cloudevents/spec/blob/v1.0.2/amqp-protocol-binding.md).

## Features

- Implements CloudEvents AMQP 0.9.1 Protocol Binding
- Supports both structured and binary content modes
- Handles CloudEvents attributes mapping to AMQP message properties
- Provides encoding and decoding of CloudEvents to/from AMQP messages

## Installation

```bash
go get github.com/your-repo/cloudevents/protocol/amqp091/v3
```

## Usage

### Creating an AMQP Protocol Instance

```go
protocol := amqp091.NewProtocol(
    // AMQP connection configuration
)
```

### Sending CloudEvents

```go
// Create a new CloudEvent
event := cloudevents.NewEvent()
event.SetID("example-id")
event.SetSource("example-source")
event.SetType("example.type")
event.SetData(cloudevents.ApplicationJSON, map[string]string{
    "hello": "world",
})

// Send the event
err := protocol.Send(context.Background(), event)
```

### Receiving CloudEvents

```go
// Start receiving events
err := protocol.StartReceiver(context.Background(), func(event cloudevents.Event) {
    // Handle the received event
})
```

## Configuration

The AMQP protocol binding can be configured with the following options:

- Connection URL
- Exchange settings
- Queue settings
- Content mode (structured/binary)
- Message persistence
- Quality of Service settings