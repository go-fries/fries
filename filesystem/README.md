# Filesystem

Filesystem provides a common logical-path storage contract for local files,
Amazon S3, and Alibaba Cloud OSS.

The base `Driver` API supports streaming reads and writes, deletion, metadata,
and paginated listing. Backend-specific features such as hard links, symbolic
links, and real directory management are exposed through optional capability
interfaces.

## Installation

```bash
go get github.com/go-fries/fries/filesystem/v4
go get github.com/go-fries/fries/filesystem/local/v4
```

## Logical paths

Paths are relative and slash-separated. Use `.` for the logical root. Leading
slashes, trailing slashes, empty path elements, `.`, `..`, and backslashes are
rejected.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-fries/fries/filesystem/local/v4"
	"github.com/go-fries/fries/filesystem/v4"
)

func main() {
	ctx := context.Background()

	driver, err := local.New("./storage")
	if err != nil {
		log.Fatal(err)
	}
	storage := filesystem.NewRepository(driver)

	if err := storage.WriteFile(
		ctx,
		"example.txt",
		[]byte("Hello, Filesystem!"),
		filesystem.PutOptions{},
	); err != nil {
		log.Fatal(err)
	}

	content, err := storage.ReadFile(ctx, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File content: %s\n", content)

	exists, err := storage.Exists(ctx, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Exists: %v\n", exists)
}
```

Amazon S3 and Alibaba Cloud OSS drivers are created with `s3.New(...)` and
`oss.New(...)`. Both accept `WithRoot(...)` to place logical paths below a
bucket prefix.

## Optional capabilities

Drivers may additionally implement:

- `filesystem.Copier`
- `filesystem.Mover`
- `filesystem.Linker`
- `filesystem.Symlinker`
- `filesystem.DirectoryManager`

`Repository.Copy` and `Repository.Move` prefer native driver capabilities and
fall back to portable streaming operations when those capabilities are absent.
