package slices

import (
	"math/rand"

	"github.com/go-fries/fries/v3/constraints"
)

// Map returns a new slice containing the results of applying the given function to each item of a given slice.
//
//	Map([]int{1, 2, 3}, func(i, _ int) int { return i * 2 }) // []int{2, 4, 6}
//	Map([]string{"a", "b", "c"}, func(s string, _ int) string { return s + "!" }) // []string{"a!", "b!", "c!"}
func Map[S ~[]E, E, R any](s S, fn func(E, int) R) []R {
	result := make([]R, 0, len(s))
	for i, item := range s {
		result = append(result, fn(item, i))
	}
	return result
}

// KeyMap returns a new map containing the results of applying the given function to each item of a given slice.
//
//	KeyMap([]int{1, 2, 3}, func(i, _ int) (int, int) { return i, i * 2 }) // map[int]int{1: 2, 2: 4, 3: 6}
//	KeyMap([]string{"a", "b", "c"}, func(s string, _ int) (string, string) {
//		return s, s + "!"
//	}) // map[string]string{"a": "a!", "b": "b!", "c": "c!"}
func KeyMap[S ~[]E, E, R any, K comparable](s S, fn func(E, int) (K, R)) map[K]R {
	result := make(map[K]R)
	for i, item := range s {
		key, value := fn(item, i)
		result[key] = value
	}
	return result
}

// Each calls the given function for each item of a given slice.
//
//	Each([]int{1, 2, 3}, func(i int, _ int) { fmt.Println(i) }) // 1\n2\n3\n
func Each[S ~[]E, E any](s S, fn func(E, int)) {
	for i, item := range s {
		fn(item, i)
	}
}

// Prepend adds the given items to the beginning of a given slice and returns the result.
//
//	Prepend([]int{1, 2, 3}, 4, 5) // []int{4, 5, 1, 2, 3}
func Prepend[S ~[]E, E any](s S, items ...E) S {
	return append(items, s...)
}

// Append adds the given items to the end of a given slice and returns the result.
//
//	Append([]int{1, 2, 3}, 4, 5) // []int{1, 2, 3, 4, 5}
func Append[S ~[]E, E any](s S, items ...E) S {
	return append(s, items...)
}

// Filter returns a new slice containing the items of a given slice that satisfy the given predicate function.
//
//	Filter([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // []int{2, 3}
//	Filter([]string{"a", "b", "c"}, func(s string, _ int) bool { return s != "b" }) // []string{"a", "c"}
func Filter[S ~[]E, R []E, E any](s S, fn func(E, int) bool) R {
	var result R
	for i, item := range s {
		if fn(item, i) {
			result = append(result, item)
		}
	}
	return result
}

// Reduce applies the given function against an accumulator
// and each element in the slice to reduce it to a single value.
//
//	Reduce([]int{1, 2, 3}, func(acc, i, _ int) int { return acc + i }, 0) // 6
//	Reduce([]string{"a", "b", "c"}, func(acc, s string, _ int) string { return acc + s }, "") // "abc"
//	Reduce([]int{1, 2, 3}, func(acc, i, _ int) int { return acc + i }, 5) // 11
func Reduce[S ~[]E, E, R any](s S, fn func(R, E, int) R, defaults ...R) R {
	var result R
	if len(defaults) > 0 {
		result = defaults[0]
	}
	for i, item := range s {
		result = fn(result, item, i)
	}
	return result
}

// Reverse returns a new slice containing the items of a given slice in reverse order.
//
//	Reverse([]int{1, 2, 3}) // []int{3, 2, 1}
func Reverse[S ~[]E, E any](s S) S {
	result := make(S, len(s))
	for i := 0; i < len(s); i++ {
		result[i] = s[len(s)-1-i]
	}
	return result
}

// Concat returns a new slice containing the items of all given slices.
//
//	Concat([]int{1, 2}, []int{3, 4}, []int{5, 6}) // []int{1, 2, 3, 4, 5, 6}
func Concat[S ~[]E, E any](slices ...S) S {
	var result S
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

// IsEmpty returns true if a given slice is empty, false otherwise.
//
//	IsEmpty([]int{}) // true
//	IsEmpty([]int{1, 2, 3}) // false
func IsEmpty[S ~[]E, E any](s S) bool {
	return len(s) == 0
}

// IsNotEmpty returns true if a given slice is not empty, false otherwise.
//
//	IsNotEmpty([]int{}) // false
//	IsNotEmpty([]int{1, 2, 3}) // true
func IsNotEmpty[S ~[]E, E any](s S) bool {
	return len(s) > 0
}

// Contains returns true if a given slice contains the given item, false otherwise.
//
//	Contains([]int{1, 2, 3}, 2) // true
//	Contains([]string{"a", "b", "c"}, "d") // false
func Contains[S ~[]E, E comparable](s S, e E) bool {
	for _, item := range s {
		if item == e {
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first occurrence of the given item in a given slice, or -1 if the item is not found.
//
//	IndexOf([]int{1, 2, 3}, 2) // 1
//	IndexOf([]string{"a", "b", "c"}, "d") // -1
func IndexOf[S ~[]E, E comparable](s S, e E) int {
	for i, item := range s {
		if item == e {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence of
// the given item in a given slice, or -1 if the item is not found.
//
//	LastIndexOf([]int{1, 2, 3, 2}, 2) // 3
//	LastIndexOf([]string{"a", "b", "c"}, "d") // -1
func LastIndexOf[S ~[]E, E comparable](s S, e E) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == e {
			return i
		}
	}
	return -1
}

// Unique returns a new slice containing the unique items of a given slice.
//
//	Unique([]int{1, 2, 2, 3, 3, 3}) // []int{1, 2, 3}
//	Unique([]string{"a", "b", "b", "c", "c", "c"}) // []string{"a", "b", "c"}
func Unique[S ~[]E, E comparable](s S) S {
	var result S
	seeds := make(map[E]struct{})
	for _, item := range s {
		if _, ok := seeds[item]; !ok {
			seeds[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// UniqueBy returns a new slice containing the unique items of
// a given slice based on the given function.
//
//	UniqueBy([]string{"apple", "apple2", "cherry"}, func(s string, _ int) string {
//		return s[:1]
//	}) // []string{"apple", "cherry"}
func UniqueBy[S ~[]E, E any, K comparable](s S, fn func(E, int) K) S {
	var result S
	seeds := make(map[K]struct{})
	for i, item := range s {
		key := fn(item, i)
		if _, ok := seeds[key]; !ok {
			seeds[key] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// Difference returns a new slice containing the items of a given
// slice that are not present in the other given slice.
//
//	Difference([]int{1, 2, 3}, []int{2, 3, 4}) // []int{1}
//	Difference([]string{"a", "b", "c"}, []string{"b", "c", "d"}) // []string{"a"}
func Difference[S ~[]E, E comparable](s1, s2 S) S {
	var result S
	for _, item := range s1 {
		if !Contains(s2, item) {
			result = append(result, item)
		}
	}
	return result
}

// Intersect returns a new slice containing the items of a given
// slice that are also present in the other given slice.
//
//	Intersect([]int{1, 2, 3}, []int{2, 3, 4}) // []int{2, 3}
//	Intersect([]string{"a", "b", "c"}, []string{"b", "c", "d"}) // []string{"b", "c"}
func Intersect[S ~[]E, E comparable](s1, s2 S) S {
	var result S
	for _, item := range s1 {
		if Contains(s2, item) {
			result = append(result, item)
		}
	}
	return result
}

// Only returns a new slice only containing the items of a given
//
//	Only([]int{1, 2, 3}, 2, 3) // []int{2, 3}
//	Only([]string{"a", "b", "c"}, "b", "c") // []string{"b", "c"}
func Only[S ~[]E, E comparable](s S, items ...E) S {
	var result S
	for _, item := range s {
		if Contains(items, item) {
			result = append(result, item)
		}
	}
	return result
}

// Without returns a new slice without the items of a given slice.
//
//	Without([]int{1, 2, 3}, 2, 3) // []int{1}
//	Without([]string{"a", "b", "c"}, "b", "c") // []string{"a"}
func Without[S ~[]E, E comparable](s S, items ...E) S {
	var result S
	for _, item := range s {
		if !Contains(items, item) {
			result = append(result, item)
		}
	}
	return result
}

// Remove returns a new slice without the given items.
// Alias for Without.
//
//	Remove([]int{1, 2, 3}, 2, 3) // []int{1}
//	Remove([]string{"a", "b", "c"}, "b", "c") // []string{"a"}
func Remove[S ~[]E, E comparable](s S, items ...E) S {
	return Without(s, items...)
}

// Partition returns two new slices, the first containing
// the items of a given slice that satisfy the given predicate function,
// and the second containing the items that do not satisfy the predicate function.
//
//	Partition([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // ([]int{2, 3}, []int{1})
func Partition[S ~[]E, E any](s S, fn func(E, int) bool) (yes, no S) {
	for i, item := range s {
		if fn(item, i) {
			yes = append(yes, item)
		} else {
			no = append(no, item)
		}
	}
	return
}

// Chunk returns a new slice containing the items of a given slice chunked into smaller slices of a given size.
//
//	Chunk([]int{1, 2, 3, 4, 5}, 2) // [][]int{{1, 2}, {3, 4}, {5}}
//	Chunk([]string{"a", "b", "c", "d", "e"}, 2) // [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
func Chunk[S ~[]E, E any](s S, size int) (result []S) {
	length := len(s)
	for i := 0; i < length; i += size {
		end := i + size
		if end > length {
			end = length
		}
		result = append(result, s[i:end])
	}
	return
}

// GroupBy returns a new map containing the items of a given slice grouped by the result of the given function.
//
//	GroupBy([]int{1, 2, 3, 4, 5}, func(i int, _ int) int { return i % 2 }) // map[int][]int{0: {2, 4}, 1: {1, 3, 5}}
func GroupBy[S ~[]E, E any, K comparable](s S, fn func(E, int) K) map[K]S {
	result := make(map[K]S)
	for i, item := range s {
		key := fn(item, i)
		result[key] = append(result[key], item)
	}
	return result
}

// First returns the first item of a given slice, or a zero value and false if the slice is empty.
//
//	First([]int{1, 2, 3}) // 1, true
//	First([]int{}) // 0, false
func First[S ~[]E, E any](s S) (E, bool) {
	if len(s) == 0 {
		var zero E
		return zero, false
	}
	return s[0], true
}

// Last returns the last item of a given slice, or a zero value and false if the slice is empty.
//
//	Last([]int{1, 2, 3}) // 3, true
//	Last([]int{}) // 0, false
func Last[S ~[]E, E any](s S) (E, bool) {
	if len(s) == 0 {
		var zero E
		return zero, false
	}
	return s[len(s)-1], true
}

// Find returns the first item of a given slice that satisfies
// the given predicate function, or a zero value and false,
// if no item satisfies the predicate function.
//
//	Find([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // 2, true
//	Find([]int{1, 2, 3}, func(i int, _ int) bool { return i > 3 }) // 0, false
func Find[S ~[]E, E any](s S, fn func(E, int) bool) (E, bool) {
	for i, item := range s {
		if fn(item, i) {
			return item, true
		}
	}
	var zero E
	return zero, false
}

// FindLast returns the last item of a given slice that satisfies
// the given predicate function, or a zero value and false,
// if no item satisfies the predicate function.
//
//	FindLast([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // 3, true
//	FindLast([]int{1, 2, 3}, func(i int, _ int) bool { return i > 3 }) // 0, false
func FindLast[S ~[]E, E any](s S, fn func(E, int) bool) (E, bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if fn(s[i], i) {
			return s[i], true
		}
	}
	var zero E
	return zero, false
}

// Index returns the index of the first item of a given slice that satisfies
// the given predicate function, or -1 and false if no item satisfies the predicate function.
//
//	Index([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // 1, true
//	Index([]int{1, 2, 3}, func(i int, _ int) bool { return i > 3 }) // -1, false
func Index[S ~[]E, E any](s S, fn func(E, int) bool) (int, bool) {
	for i, item := range s {
		if fn(item, i) {
			return i, true
		}
	}
	return -1, false
}

// LastIndex returns the index of the last item of a given slice that satisfies
// the given predicate function, or -1 and false if no item satisfies the predicate function.
//
//	LastIndex([]int{1, 2, 3}, func(i int, _ int) bool { return i > 1 }) // 2, true
//	LastIndex([]int{1, 2, 3}, func(i int, _ int) bool { return i > 3 }) // -1, false
func LastIndex[S ~[]E, E any](s S, fn func(E, int) bool) (int, bool) {
	for i := len(s) - 1; i >= 0; i-- {
		if fn(s[i], i) {
			return i, true
		}
	}
	return -1, false
}

// Fill returns a given slice with the given value.
//
//	Fill([]int{1, 2, 3}, 0) // []int{0, 0, 0}
//	Fill(make([]int, 3), 1) // []int{1, 1, 1}
func Fill[S ~[]E, E any](s S, value E) S {
	for i := range s {
		s[i] = value
	}
	return s
}

// Random returns a random item of a given slice.
//
//	Random([]int{1, 2, 3}) // 2
//	Random([]string{"a", "b", "c"}) // "b"
func Random[S ~[]E, E any](s S) E {
	return s[rand.Intn(len(s))]
}

// Shuffle returns a new slice containing the items of a given slice shuffled.
//
//	Shuffle([]int{1, 2, 3}) // []int{3, 1, 2}
func Shuffle[S ~[]E, E any](s S) S {
	result := make(S, len(s))
	copy(result, s)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Min returns the minimum item of a given slice.
//
//	Min([]int{1, 2, 3}) // 1
//	Min([]string{"a", "b", "c"}) // "a"
func Min[S ~[]E, E constraints.Ordered](s S) E {
	m := s[0]
	for _, item := range s {
		if item < m {
			m = item
		}
	}
	return m
}

// Max returns the maximum item of a given slice.
//
//	Max([]int{1, 2, 3}) // 3
//	Max([]string{"a", "b", "c"}) // "c"
func Max[S ~[]E, E constraints.Ordered](s S) E {
	m := s[0]
	for _, item := range s {
		if item > m {
			m = item
		}
	}
	return m
}

// Sum returns the sum of the items of a given slice.
//
//	Sum([]int{1, 2, 3}) // 6
func Sum[S ~[]E, E constraints.Numeric](s S) E {
	var sum E
	for _, item := range s {
		sum += item
	}
	return sum
}

// Length returns the length of a given slice.
//
//	Length([]int{1, 2, 3}) // 3
//	Length([]string{"a", "b", "c"}) // 3
func Length[S ~[]E, E any](s S) int {
	return len(s)
}
