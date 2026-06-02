# Coordinator

## Usage

```go
package main

import (
	"fmt"
	"sync"

	"github.com/go-fries/fries/support/v3/coordinator"
)

func main() {
	foo := coordinator.NewCoordinator()
	bar := coordinator.NewCoordinator()

	var wg sync.WaitGroup
	wg.Add(3) //nolint:gomnd

	go func() {
		defer wg.Done()
		<-foo.Done()
		fmt.Println("foo")
	}()

	go func() {
		defer wg.Done()
		<-foo.Done()
		fmt.Println("foo 2")
	}()

	go func() {
		defer wg.Done()
		<-bar.Done()
		fmt.Println("bar")
	}()

	foo.Close()
	bar.Close()

	wg.Wait()
}

```
