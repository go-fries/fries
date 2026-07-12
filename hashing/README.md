# Hashing

`hashing` provides small, reusable helpers for deterministic content digests.
It accepts any algorithm exposed through Go's standard `hash.Hash` interface
and supports byte slices, strings, streams, and files.

This module is intended for checksums, cache keys, content fingerprints, and
compatibility protocols. It is not a password-hashing or message-
authentication API.

## Installation

```bash
go get github.com/go-fries/fries/hashing/v4
```

Install an algorithm helper only when it is needed:

```bash
go get github.com/go-fries/fries/hashing/md5/v4
```

## General algorithms

Create a reusable hasher from any standard-library or third-party constructor:

```go
package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"

	"github.com/go-fries/fries/hashing/v4"
)

func main() {
	hasher := hashing.New(sha256.New)

	digest := hasher.SumString("hello")
	fmt.Println(digest.Hex())
	fmt.Println(digest.Base64())

	streamDigest, err := hasher.SumReader(strings.NewReader("hello"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(digest.Equal(streamDigest))
}
```

A `Hasher` creates a fresh `hash.Hash` for each operation and can therefore be
shared by concurrent callers when its constructor is concurrency-safe.

## Files

```go
digest, err := hashing.New(sha256.New).SumFile("archive.tar.gz")
if err != nil {
	return err
}
fmt.Println(digest.Hex())
```

## Parsing stored digests

```go
expected, err := hashing.ParseHex(storedHex)
if err != nil {
	return err
}

actual := hashing.New(sha256.New).Sum(payload)
if !actual.Equal(expected) {
	return errors.New("checksum mismatch")
}
```

`Digest.Equal` compares equal-length digest contents in constant time.

## MD5 compatibility

```go
import "github.com/go-fries/fries/hashing/md5/v4"

checksum := md5.SumString("legacy payload").Hex()
```

MD5 is cryptographically broken. Use it only where an existing checksum or
legacy protocol explicitly requires MD5. Do not use MD5 for passwords,
signatures, certificates, or other security-sensitive purposes.

## Migrating from the previous API

| Before | After |
| --- | --- |
| `hashing.MD5.New()` | `md5.New()` |
| `hasher.Make(value)` | `hasher.SumString(value).Hex()` |
| `hasher.MustMake(value)` | `hasher.SumString(value).Hex()` |
| `hasher.Check(value, encoded)` | Parse with `hashing.ParseHex`, then call `Equal` |
| `hashing.Register(...)` | Pass the algorithm constructor to `hashing.New(...)` |

The enum and global registry were removed. Algorithms are now ordinary
constructor dependencies, matching Go's standard hashing model.
