# Jet —— Hyperf RPC Client

## Introduction

This is an RPC client compatible with [Hyperf](https://github.com/hyperf/hyperf), supporting remote procedure calls (RPC) via JSON-RPC. Support for `middleware`, `ID generation`, `path generation`, and other features.

## Usage Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-fries/fries/hyperf/jet/v3"
)

func main() {
	// create transporter
	transport, err := jet.NewHTTPTransporter(
		jet.WithHTTPTransporterAddr("http://localhost:8080/"), // http server address
	)
	if err != nil {
		panic(err)
	}

	// create client
	client, err := jet.NewClient(
		jet.WithService("Example/User/MoneyService"), // service name
		jet.WithTransporter(transport),
		jet.WithMiddleware(recovery(), logger()), // with middleware(if you need)
	)
	if err != nil {
		panic(err)
	}

	// use middleware(if you need)
	client.Use(recovery(), logger()) // use middleware(if you need)

	// call service
	var balance float64
	if err := client.Invoke(context.Background(), "getBalance", []any{1006}, &balance); err != nil {
		panic(err)
	}

	log.Println(balance)

	// call service with middleware(if you need)
	if err := client.Invoke(context.Background(), "getBalance", []any{1006}, &balance, recovery(), logger()); err != nil {
		panic(err)
	}
}

func recovery() jet.Middleware {
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, client *jet.Client, name string, request any) (response any, err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Println("recovered:", r)
					err = fmt.Errorf("%v", r)
				}
			}()
			return next(ctx, client, name, request)
		}
	}
}

func logger() jet.Middleware {
	return func(next jet.Handler) jet.Handler {
		return func(ctx context.Context, client *jet.Client, name string, request any) (response any, err error) {
			log.Println("service", client.GetService(), "name:", name, "request:", request, "response:", response, "error:", err)
			return next(ctx, client, name, request)
		}
	}
}
```