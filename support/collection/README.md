# Collection

## Example

```go
package main

import "github.com/go-kratos-ecosystem/components/v2/collection"

func main() {
	// int
	c := collection.New([]int{1, 2, 3})
	c.Add(4) //nolint:gomnd
	c.Items()
	c.All()
	c.Len()
	c.Map(func(i int, _ int) int {
		return i * 2
	})
	c.Filter(func(v int, _ int) bool {
		return v > 4
	})
	c.Where(func(v int, _ int) bool {
		return v > 4
	})
	c.Reduce(func(a, b int) int {
		return a + b
	})
	// ....

	// string
	c2 := collection.New([]string{"a", "b", "c"})
	c2.Add("d")

	// and so on
}
```