# JSON-RPC Client

A lightweight JSON-RPC 2.0 client implementation in Go.

## Installation

```bash
go get github.com/go-fries/fries/jsonrpc/v3
```

## Quick Start

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/go-fries/fries/jsonrpc/v3"
)

// Logger middleware example
func Logger() jsonrpc.Middleware {
    return func(next jsonrpc.Handler) jsonrpc.Handler {
        return func(ctx context.Context, namespace string, req *jsonrpc.Request) (resp *jsonrpc.Response, err error) {
            defer func(start time.Time) {
                log.Println("[JSON-RPC]", namespace, req.Method, "took", time.Since(start), "error:", err)
            }(time.Now())
            return next(ctx, namespace, req)
        }
    }
}

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Age  int    `json:"age"`
}

func main() {
    // Create transport
    transport := jsonrpc.NewHTTPTransport("https://jsonrpc.example.com")

    // Create client with middleware
    client := jsonrpc.NewClient(transport, jsonrpc.WithMiddlewares(Logger()))

    // Invoke remote method
    var user User
    resp, err := client.Invoke(context.Background(), &user, "getUser", 12345)
    if err != nil {
        log.Fatal(err)
    }

    log.Println("User:", user)

    // Use namespace
    nsClient := client.Namespace("admin")
    var result string
    resp, err = nsClient.Invoke(context.Background(), &result, "sayHello", "World")
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Result:", result)
}
```

## Components

### Client

The main client interface for invoking JSON-RPC methods. Supports middleware chaining and namespace scoping.

```go
client := jsonrpc.NewClient(transport)
client.Invoke(ctx, &result, "method", args...)
```

### Transport

Defines how requests are sent to the server. Built-in HTTP transport included.

```go
transport := jsonrpc.NewHTTPTransport("https://api.example.com")
```

### Middleware

Intercept and process requests (e.g., logging, authentication).

```go
client.Use(loggingMiddleware, authMiddleware)
```

### Namespace

Organize methods under different contexts.

```go
adminClient := client.Namespace("admin")
```

## Options

**Client Options:**
- `WithMiddlewares()` - Add middlewares
- `WithIDGenerator()` - Custom ID generator
- `WithCodec()` - Custom codec

**Transport Options:**
- `WithHTTPTransportClient()` - Custom HTTP client
- `WithHTTPTransportUserAgent()` - Set User-Agent header