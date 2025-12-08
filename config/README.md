# Config

A simple, type-safe utility for propagating configuration values through `context.Context`.

## Installation

```bash
go get github.com/go-fries/fries/config/v3
```

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/go-fries/fries/config/v3"
)

type AppConfig struct {
	Debug bool
	Port  int
}

func main() {
	ctx := context.Background()

	// Store config in context
	cfg := AppConfig{Debug: true, Port: 8080}
	ctx = config.NewContext(ctx, cfg)

	// Retrieve config from context later
	if retrieved, ok := config.FromContext[AppConfig](ctx); ok {
		fmt.Printf("Debug: %v, Port: %d\n", retrieved.Debug, retrieved.Port)
	} else {
		fmt.Println("Config not found in context")
	}
}
```

```
