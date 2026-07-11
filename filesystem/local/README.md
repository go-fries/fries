# Local Filesystem

The Local Filesystem driver stores logical filesystem paths below a local root
directory. It implements the core filesystem API and supports moves, hard
links, symbolic links, and real directory management.

## Installation

```bash
go get github.com/go-fries/fries/filesystem/v4
go get github.com/go-fries/fries/filesystem/local/v4
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-fries/fries/filesystem/local/v4"
	"github.com/go-fries/fries/filesystem/v4"
)

func main() {
	ctx := context.Background()

	if err := os.MkdirAll("./storage", 0o755); err != nil {
		log.Fatal(err)
	}
	driver, err := local.New("./storage")
	if err != nil {
		log.Fatal(err)
	}
	storage := filesystem.NewRepository(driver)

	if err := driver.MakeDirectory(ctx, "documents"); err != nil {
		log.Fatal(err)
	}
	if err := storage.WriteFile(
		ctx,
		"documents/example.txt",
		[]byte("local content"),
		filesystem.PutOptions{},
	); err != nil {
		log.Fatal(err)
	}

	page, err := storage.ListFiles(ctx, "documents", filesystem.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range page.Entries {
		fmt.Println(entry.Path)
	}

	if err := driver.Symlink(
		ctx,
		"documents/example.txt",
		"documents/example-link.txt",
	); err != nil {
		log.Fatal(err)
	}
}
```

## Behavior

- All paths are resolved below the configured root.
- Internal symbolic links are supported, but links cannot be followed outside
  the root.
- `ListFiles` includes regular files and internal symbolic links to regular
  files. Directories and special filesystem nodes are excluded.
- Directory creation and deletion are available directly on `*local.Filesystem`.
