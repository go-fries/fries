package debug

import (
	"fmt"
	"os"
	"testing"
)

func TestDump(_ *testing.T) {
	Dump("foo", []byte("1234567890"), &struct {
		Name string
	}{
		Name: "foo",
	}, func() {
		panic("foo")
	})

	fmt.Println(Sdump("foo", []byte("1234567890"), &struct {
		Name string
	}{
		Name: "foo",
	}))

	Fdump(os.Stdout, "foo", []byte("1234567890"), &struct {
		Name string
	}{
		Name: "foo",
	})
}
