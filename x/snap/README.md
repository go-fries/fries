# Snap

The `snap` package in Go provides a thread-safe mechanism to manage and periodically refresh cached values using a custom refresh function. 

It supports configurable refresh intervals and ensures that the value is always up-to-date without manual intervention. 

This package is ideal for scenarios where you need to maintain a fresh state with minimal overhead.

## Features

- **Thread-safe:** Ensures safe access and updates in concurrent scenarios.
- **Configurable Refresh Interval:** Allows you to specify how often the value should be refreshed.
- **Automatic Refresh:** Automatically updates the cached value using a provided refresh function.


## Usage

```go
package main

import (
	"fmt"
	"time"

	"github.com/go-kratos-ecosystem/components/v2/snap"
)

type Config struct {
	Name string
}

func main() {
	s := snap.New(func() (*Config, error) {
		// get config from remote/datastore/...
		return &Config{
			Name: "test",
		}, nil
	}, snap.Interval[*Config](1000*time.Second))

	fmt.Print(s.Get()) // &{test} <nil>&
	fmt.Print(s.Get()) // &{test} <nil>&
	// after 1000 seconds
	fmt.Print(s.Get()) // &{test} <nil>& after refresh

	// refresh immediately
	s.Refresh()
}

```