package collection

import (
	"sort"

	"github.com/go-fries/fries/v3/debug"
)

type Collection[T comparable] struct { //nolint:gofumpt
	items []T
}

func New[T comparable](items []T) *Collection[T] {
	return &Collection[T]{
		items: items,
	}
}

func (c *Collection[T]) Add(items ...T) {
	c.items = append(c.items, items...)
}

func (c *Collection[T]) Items() []T {
	return c.items
}

func (c *Collection[T]) All() []T {
	return c.Items()
}

func (c *Collection[T]) Len() int {
	return len(c.items)
}

func (c *Collection[T]) Map(fn func(T, int) T) *Collection[T] {
	var items []T //nolint:prealloc
	for i, item := range c.items {
		items = append(items, fn(item, i))
	}
	return New(items)
}

func (c *Collection[T]) Filter(fn func(T, int) bool) *Collection[T] {
	var items []T //nolint:prealloc
	for i, item := range c.items {
		if fn(item, i) {
			items = append(items, item)
		}
	}
	return New(items)
}

func (c *Collection[T]) Where(fn func(T, int) bool) *Collection[T] {
	return c.Filter(fn)
}

func (c *Collection[T]) Reduce(fn func(T, T) T) T {
	var result T
	for _, item := range c.items {
		result = fn(result, item)
	}
	return result
}

func (c *Collection[T]) Each(fn func(T, int)) {
	for i, item := range c.items {
		fn(item, i)
	}
}

func (c *Collection[T]) Find(fn func(T, int) bool) (T, bool) {
	for i, item := range c.items {
		if fn(item, i) {
			return item, true
		}
	}

	var zero T
	return zero, false
}

func (c *Collection[T]) First() (T, bool) {
	if c.IsEmpty() {
		var zero T
		return zero, false
	}
	return c.items[0], true
}

func (c *Collection[T]) Index(fn func(T, int) bool) (int, bool) {
	for i, item := range c.items {
		if fn(item, i) {
			return i, true
		}
	}
	return -1, false
}

func (c *Collection[T]) Contains(item T) bool {
	for _, i := range c.items {
		if i == item {
			return true
		}
	}
	return false
}

func (c *Collection[T]) Has(item T) bool {
	return c.Contains(item)
}

func (c *Collection[T]) Exists(item T) bool {
	return c.Contains(item)
}

func (c *Collection[T]) Missing(item T) bool {
	return !c.Contains(item)
}

func (c *Collection[T]) IsEmpty() bool {
	return c.Len() == 0
}

func (c *Collection[T]) IsNotEmpty() bool {
	return !c.IsEmpty()
}

func (c *Collection[T]) Last() (T, bool) {
	if c.IsEmpty() {
		var zero T
		return zero, false
	}
	return c.items[c.Len()-1], true
}

func (c *Collection[T]) Unique() *Collection[T] {
	var items []T //nolint:prealloc
	seen := make(map[T]struct{})
	for _, item := range c.items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		items = append(items, item)
	}
	return New(items)
}

func (c *Collection[T]) Reverse() *Collection[T] {
	var items []T //nolint:prealloc
	for i := c.Len() - 1; i >= 0; i-- {
		items = append(items, c.items[i])
	}
	return New(items)
}

func (c *Collection[T]) SortBy(fn func(T, T) bool) *Collection[T] {
	var items []T //nolint:prealloc
	items = append(items, c.items...)
	sort.Slice(items, func(i, j int) bool {
		return fn(items[i], items[j])
	})
	return New(items)
}

func (c *Collection[T]) Dump() {
	debug.Dump(c.items)
}

func (c *Collection[T]) When(condition bool, fns ...func(*Collection[T])) *Collection[T] {
	if condition {
		for _, fn := range fns {
			if fn != nil {
				fn(c)
			}
		}
	}
	return c
}

func (c *Collection[T]) Unless(condition bool, fns ...func(*Collection[T])) *Collection[T] {
	return c.When(!condition, fns...)
}
