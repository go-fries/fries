package support

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// Repeat runs the given function `times` times or until an error is returned.
//
//	Repeat(func() error { fmt.Println("hello"); return nil }, 3) => prints hello 3 times and returns nil
//	Repeat(func() error { return fmt.Errorf("error") }, 3) => returns error
func Repeat(fn func() error, times int) error {
	for range times {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

// ErrorIf returns an error if the condition is true.
//
//	ErrorIf(true, "error") => error
//	ErrorIf(false, "error") => nil
//	ErrorIf(true, "error %s", "with value") => error with value
func ErrorIf(condition bool, format string, a ...any) error {
	if condition {
		return fmt.Errorf(format, a...)
	}
	return nil
}

// PanicIf panics if the condition is true.
//
//	PanicIf(true, "error") => panic("error")
//	PanicIf(false, "error") => nil
//	PanicIf(true, "error %s", "with value") => panic("error with value")
func PanicIf(condition bool, format string, a ...any) {
	if condition {
		panic(fmt.Sprintf(format, a...))
	}
}

// Pipe is a function that takes a value and returns a value
//
//	Pipe(m1, m2, m3)(value) => m3(m2(m1(value)))
func Pipe[T any](fns ...func(T) T) func(T) T {
	return func(v T) T {
		for _, fn := range fns {
			v = fn(v)
		}
		return v
	}
}

// PipeWithErr is a function that takes a value and returns a value and an error
//
//	PipeWithErr(m1, m2, m3)(value) => m3(m2(m1(value)))
func PipeWithErr[T any](fns ...func(T) (T, error)) func(T) (T, error) {
	var err error
	return func(v T) (T, error) {
		for _, fn := range fns {
			if v, err = fn(v); err != nil {
				return v, err
			}
		}
		return v, nil
	}
}

// Chain is a reverse Pipe
//
//	Chain(m1, m2, m3)(value) => m1(m2(m3(value)))
func Chain[T any](fns ...func(T) T) func(T) T {
	return func(v T) T {
		for _, fn := range slices.Backward(fns) {
			if fn != nil {
				v = fn(v)
			}
		}
		return v
	}
}

// ChainWithErr is a reverse PipeWithErr
//
//	ChainWithErr(m1, m2, m3)(value) => m1(m2(m3(value)))
func ChainWithErr[T any](fns ...func(T) (T, error)) func(T) (T, error) {
	var err error
	return func(v T) (T, error) {
		for _, fn := range slices.Backward(fns) {
			if fn != nil {
				if v, err = fn(v); err != nil {
					return v, err
				}
			}
		}
		return v, nil
	}
}

// Scan sets the value of dest to the value of src.
//
//	var foo string
//	Scan("bar", &foo) // foo == "bar"
//
//	var bar struct {A string}
//	Scan(struct{A string}{"foo"}, &bar) // bar == struct{A string}{"foo"}
func Scan(src, dest any) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, dest)
}

// Once returns a function that calls the given function only once.
//
//	once := Once(func() { fmt.Println("hello") })
//	once() // prints hello
//	once() // does nothing
func Once(fn func()) func() {
	var once sync.Once
	return func() {
		once.Do(fn)
	}
}

// When calls the given callback with the given value if the condition is true.
func When(condition bool, callback func()) {
	if condition {
		callback()
	}
}

// Must is a helper function that panics if the error is not nil.
//
//	Must(strconv.Atoi("123")) // returns 123
//	Must(strconv.Atoi("abc")) // panics with error
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

// Ignore is a helper function that returns the value and ignores the error.
//
//	Ignore(strconv.Atoi("123")) // returns 123
//	Ignore(strconv.Atoi("abc")) // returns 0
func Ignore[T any](v T, _ error) T {
	return v
}
