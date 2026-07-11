# Amazon S3 Filesystem

The Amazon S3 Filesystem driver stores logical filesystem paths as objects in
an S3 bucket. It implements the core filesystem API and provides native copy
and move capabilities.

## Installation

```bash
go get github.com/go-fries/fries/filesystem/v4
go get github.com/go-fries/fries/filesystem/s3/v4
go get github.com/aws/aws-sdk-go-v2/config
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-fries/fries/filesystem/v4"
	filesystemS3 "github.com/go-fries/fries/filesystem/s3/v4"
)

func main() {
	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}
	client := awss3.NewFromConfig(cfg)
	driver := filesystemS3.New(
		client,
		"my-bucket",
		filesystemS3.WithRoot("application"),
	)
	storage := filesystem.NewRepository(driver)

	if err := storage.WriteFile(
		ctx,
		"documents/example.txt",
		[]byte("s3 content"),
		filesystem.PutOptions{ContentType: "text/plain"},
	); err != nil {
		log.Fatal(err)
	}

	page, err := storage.ListFiles(
		ctx,
		"documents",
		filesystem.ListOptions{Recursive: true},
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range page.Entries {
		fmt.Printf("%s (%d bytes)\n", entry.Path, entry.Size)
	}

	if err := storage.Copy(
		ctx,
		"documents/example.txt",
		"documents/example-copy.txt",
	); err != nil {
		log.Fatal(err)
	}
}
```

## Behavior

- `WithRoot` places every logical path below an object-key prefix. In the
  example, `documents/example.txt` maps to
  `application/documents/example.txt`.
- Directories are virtual prefixes. `ListFiles` returns objects only and omits
  directory markers.
- `Repository.Copy` uses S3's native server-side copy operation.
- Move is implemented as copy followed by delete and is not atomic.
