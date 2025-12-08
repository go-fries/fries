# Env

Simple utilities for managing application runtime environments (Dev, Prod, Debug, Stage).

## Installation

```bash
go get github.com/go-fries/fries/env/v3
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-fries/fries/env/v3"
)

func main() {
	// Set the current environment
	env.SetEnv(env.Dev)

	// Check the environment
	if env.IsDev() {
		fmt.Println("Running in Development mode")
	}

	if env.IsProd() {
		fmt.Println("Running in Production mode")
	}

	// Custom check
	if env.Is(env.Dev, env.Debug) {
		fmt.Println("Environment is either Dev or Debug")
	}
}
```
