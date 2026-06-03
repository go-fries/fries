# Slices

This package provides generic slice helper functions for common application
code.

## Installation

```bash
go get github.com/go-fries/fries/slices/v3
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-fries/fries/slices/v3"
)

func main() {
	values := []int{1, 2, 3}

	fmt.Println(slices.Map(values, func(v int) int { return v * 2 }))
	fmt.Println(slices.Filter(values, func(v int) bool { return v > 1 }))
	fmt.Println(slices.Sum(values))
}
```

## Helpers

- `Map`, `MapN`, `FlatMap`, `KeyBy`, `KeyByN`, and `KeyMap` transform slices into new slices or maps.
- `Each` and `EachN` iterate over slice items.
- `Prepend`, `Append`, `Concat`, and `Reverse` build derived slices.
- `Filter`, `FilterN`, `Partition`, and `PartitionN` select items by predicate.
- `Any` and `Every` test predicates across slice items.
- `Reduce` and `ReduceN` reduce a slice to a single value.
- `Unique`, `UniqueBy`, and `UniqueByN` remove duplicate items.
- `Difference`, `Intersect`, `Only`, `Without`, and `Remove` provide set-like helpers.
- `Chunk`, `GroupBy`, `GroupByN`, `CountBy`, and `CountByN` split, group, or count items.
- `First`, `Last`, `Find`, `FindN`, `FindLast`, and `FindLastN` locate items.
- `Index`, `IndexN`, `LastIndex`, and `LastIndexN` locate item indexes by predicate.
- `IndexOf` and `LastIndexOf` locate comparable items.
- `Fill`, `Random`, `Shuffle`, `Min`, `Max`, `Sum`, and `Length` provide utility operations.
