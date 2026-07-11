# Filesystem

Filesystem provides a common storage API for local filesystems, Amazon S3, and
Alibaba Cloud OSS. It uses logical paths relative to a configured root and
supports streaming reads and writes, metadata, deletion, and paginated file
listing.

The component is split into a core module and backend modules:

- [`filesystem/local`](./local) stores files below a local directory.
- [`filesystem/s3`](./s3) stores objects in Amazon S3.
- [`filesystem/oss`](./oss) stores objects in Alibaba Cloud OSS.

## Installation

Install the core module and the backend you need:

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

	err = storage.WriteFile(
		ctx,
		"example.txt",
		[]byte("Hello, Filesystem!"),
		filesystem.PutOptions{},
	)
	if err != nil {
		log.Fatal(err)
	}

	content, err := storage.ReadFile(ctx, "example.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(content))

	page, err := storage.ListFiles(ctx, ".", filesystem.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range page.Entries {
		fmt.Printf("%s (%d bytes)\n", entry.Path, entry.Size)
	}

	if err := storage.Delete(ctx, "example.txt"); err != nil {
		log.Fatal(err)
	}
}
```

## Basic behavior

- Paths are relative and slash-separated. Use `.` for the logical root.
- `Open`, `Put`, and `Delete` operate on files or objects.
- `Delete` is idempotent; deleting a missing path returns `nil`.
- `ListFiles` returns files and objects only. Set `Recursive` to include nested
  paths and follow `NextCursor` until it is empty when reading multiple pages.
- `Put` accepts an `io.Reader`. Common readers and files expose enough
  information to infer their length; other streams must set
  `PutOptions.ContentLength`.

`Repository` adds whole-file helpers, existence checks, and portable copy and
move fallbacks. Drivers may also expose optional capabilities such as native
copying, moving, links, symbolic links, and directory management.

See the backend README for setup and backend-specific behavior.
