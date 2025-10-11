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
	"net/http"
	"time"

	"github.com/go-fries/fries/jsonrpc/v3"
)

func Logger() jsonrpc.Middleware {
	return func(next jsonrpc.Handler) jsonrpc.Handler {
		return func(ctx context.Context, namespace string, req *jsonrpc.Request) (resp *jsonrpc.Response, err error) {
			defer func(start time.Time) {
				log.Println("[JSON-RPC] ", namespace, req.Method, "took", time.Since(start), "error:", err)
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
	// Create a HTTP transport with custom settings
	transport := jsonrpc.NewHTTPTransport(
		"https://jsonrpc.example.com",                       // replace with your JSON-RPC server URL
		jsonrpc.WithHTTPTransportClient(http.DefaultClient), // use custom *http.Client
		jsonrpc.WithHTTPTransportUserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"), // custom User-Agent
	)

	// Create a new JSON-RPC client with the transport and middleware
	client := jsonrpc.NewClient(
		transport,
		jsonrpc.WithMiddlewares(Logger()), // add logging middleware
		jsonrpc.WithIDGenerator(jsonrpc.DefaultIDGenerator), // use default ID generator, you can customize it if needed
		jsonrpc.WithCodec(jsonrpc.DefaultCodec),             // use default codec, you can customize it if needed
	)

	// Another way to add middleware
	client.Use(Logger())

	// Invoke a remote method
	var user User
	resp, err := client.Invoke(context.Background(), &user, "getUser", 12345) // assuming getUser is a method that takes a user ID and returns user info
	if err != nil {
		log.Fatal("Invoke error:", err)
	}

	log.Println("Response:", resp)
	log.Println("User:", user)

	// Use the client with a namespace
	nsClient := client.Namespace("myNamespace")
	var result string
	resp, err = nsClient.Invoke(context.Background(), &result, "sayHello", "World") // assuming sayHello is a method in the namespace
	if err != nil {
		log.Fatal("Namespace Invoke error:", err)
	}

	log.Println("Namespace Response:", resp)
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