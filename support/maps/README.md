# Maps

## Example

```go
package main

import (
	"fmt"

	"github.com/go-kratos-ecosystem/components/v2/maps"
)

func main() {
	m := maps.M{}

	m.Merge(map[string]any{
		"name":    "Flc",
		"age":     18, //nolint:gomnd
		"sex":     "man",
		"address": "China",
		"phone":   "123456789",
	})

	m.When(true, func(m maps.Maps) maps.Maps {
		return m.Set("first name", "wu").
			Set("last name", "Flc").
			Set("age", 19) //nolint:gomnd
	})

	fmt.Println(m.Maps())

	// output:
	// map[address:China age:19 first name:wu last name:Flc name:Flc phone:123456789 sex:man]
}
```