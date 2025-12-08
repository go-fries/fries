# Codec

Unified interface for encoding and decoding data, with support for multiple formats.

## Installation

```bash
go get github.com/go-fries/fries/codec/v3
# Install specific codecs as needed, e.g.:
go get github.com/go-fries/fries/codec/json/v3
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/go-fries/fries/codec/json/v3"
)

var j = json.Codec

func main() {
	// Marshal
	bytes, err := j.Marshal(map[string]string{
		"key": "value",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("JSON: %s\n", bytes)

	// Unmarshal
	var dest map[string]string
	err = j.Unmarshal(bytes, &dest)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decoded: %v\n", dest)
}
```