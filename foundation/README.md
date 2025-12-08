# Foundation

The core kernel of the Fries framework, responsible for managing the application lifecycle and Service Providers.

## Installation

```bash
go get github.com/go-fries/fries/foundation/v3
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-fries/fries/foundation/v3"
)

// ExampleProvider implements the foundation.Provider interface
type ExampleProvider struct{}

func (p *ExampleProvider) Bootstrap(ctx context.Context) (context.Context, error) {
	fmt.Println("Provider: Bootstrap")
	return ctx, nil
}

func (p *ExampleProvider) Terminate(ctx context.Context) (context.Context, error) {
	fmt.Println("Provider: Terminate")
	return ctx, nil
}

func main() {
	// Create the kernel
	kernel := foundation.NewKernel(
		foundation.WithHandler(foundation.HandlerFunc(func(ctx context.Context) error {
			fmt.Println("Application: Running logic...")
			return nil
		})),
	)

	// Register providers
	kernel.Register(&ExampleProvider{})

	// Run the application
	if err := kernel.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```
