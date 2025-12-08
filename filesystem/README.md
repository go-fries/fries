# Filesystem

A unified filesystem abstraction layer for Go, supporting Local, S3, and OSS storage drivers.

## Installation

```bash
go get github.com/go-fries/fries/filesystem/v3
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-fries/fries/filesystem/local/v3"
	"github.com/go-fries/fries/filesystem/v3"
)

func main() {
	ctx := context.Background()

	// Initialize the Local driver
	// You can also use s3.NewStorage(...) or oss.NewStorage(...)
	store := local.NewStorage("./storage")

	// Create the repository
	fs := filesystem.NewRepository(store)

	// Write a file
	if err := fs.Put(ctx, "example.txt", []byte("Hello, Filesystem!")); err != nil {
		log.Fatal(err)
	}

	// Read a file
	content, err := fs.Get(ctx, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File Content: %s\n", content)

	// Check if file exists
	exists, _ := fs.Has(ctx, "example.txt")
	fmt.Printf("Exists: %v\n", exists)

	// Delete the file
	// if err := fs.Destroy(ctx, "example.txt"); err != nil {
	// 	log.Fatal(err)
	// }
}
```