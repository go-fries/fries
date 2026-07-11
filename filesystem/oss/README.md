# Alibaba Cloud OSS Filesystem

The Alibaba Cloud OSS Filesystem driver stores logical filesystem paths as
objects in an OSS bucket. It implements the core filesystem API and provides
native copy and move capabilities.

## Installation

```bash
go get github.com/go-fries/fries/filesystem/v4
go get github.com/go-fries/fries/filesystem/oss/v4
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	aliyunoss "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/go-fries/fries/filesystem/v4"
	filesystemOSS "github.com/go-fries/fries/filesystem/oss/v4"
)

func main() {
	ctx := context.Background()

	cfg := aliyunoss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewEnvironmentVariableCredentialsProvider()).
		WithRegion("cn-hangzhou")
	client := aliyunoss.NewClient(cfg)
	driver := filesystemOSS.New(
		client,
		"my-bucket",
		filesystemOSS.WithRoot("application"),
	)
	storage := filesystem.NewRepository(driver)

	if err := storage.WriteFile(
		ctx,
		"documents/example.txt",
		[]byte("oss content"),
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
- `Repository.Copy` uses OSS's native server-side copy operation.
- Move is implemented as copy followed by delete and is not atomic.
