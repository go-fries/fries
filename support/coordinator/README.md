# Coordinator

## Usage

```go
package main

import (
	"fmt"
	"sync"

	"github.com/go-kratos-ecosystem/components/v2/coordinator"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(3) //nolint:gomnd

	go func() {
		defer wg.Done()
		if <-coordinator.Until("foo").Done(); true {
			fmt.Println("foo")
		}
	}()

	go func() {
		defer wg.Done()
		if <-coordinator.Until("foo").Done(); true {
			fmt.Println("foo 2")
		}
	}()

	go func() {
		defer wg.Done()
		if <-coordinator.Until("bar").Done(); true {
			fmt.Println("bar")
		}
	}()

	coordinator.Until("foo").Close()
	coordinator.Until("bar").Close()

	wg.Wait()
}

```