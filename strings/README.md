# Strings

This package provides string helper functions for common application code.

## Installation

```bash
go get github.com/go-fries/fries/strings/v3
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-fries/fries/strings/v3"
)

func main() {
	fmt.Println(strings.Is("*.example.com", "api.example.com"))
	fmt.Println(strings.MD5("abc"))
	fmt.Println(strings.Reverse("张三"))
	fmt.Println(strings.After("Hello, World!", ","))
}
```

## Helpers

- `Is` matches a string against a `*` wildcard pattern.
- `InSlice` reports whether a string exists in a slice.
- `MD5` and `SHA1` return lowercase hexadecimal hashes.
- `Reverse` reverses a string by Unicode code points.
- `Replace` replaces all occurrences of a substring.
- `Shuffle` randomly reorders characters.
- `Random` returns a random alphabetic string.
- `Len` returns the Unicode code point length.
- `IsUUID` validates UUID strings.
- `UUID` returns a new UUID string.
- `After` and `Before` split around the first occurrence of a separator.
- `SubstrCount` counts substring occurrences in a byte-indexed range.
