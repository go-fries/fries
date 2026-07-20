package ptr_test

import (
	"fmt"

	"github.com/go-fries/fries/ptr/v4"
)

func ExamplePtr() {
	name := ptr.Ptr("fries")

	fmt.Println(*name)
	// Output: fries
}

func ExampleValue() {
	name, ok := ptr.Value(ptr.Ptr("fries"))

	fmt.Println(name, ok)
	// Output: fries true
}

func ExampleOr() {
	var name *string

	fmt.Println(ptr.Or(name, "fries"))
	// Output: fries
}
