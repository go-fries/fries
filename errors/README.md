# Errors

Helper functions for creating and handling Kratos errors with standard HTTP status codes.

## Installation

```bash
go get github.com/go-fries/fries/errors/v3
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-fries/fries/errors/v3"
)

func main() {
	// Create a NotFound error
	err := errors.NotFound("user not found")
	fmt.Println(err)

	// Check error type
	if errors.IsNotFound(err) {
		fmt.Println("It was a not found error")
	}

	// Create a generic error
	err2 := errors.New(418, "I'm a teapot")
	fmt.Println(err2)
}
```
