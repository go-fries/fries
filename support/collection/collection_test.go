package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollection_Int(t *testing.T) {
	c := New([]int{1, 2, 3})
	c.Add(4)
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []int{1, 2, 3, 4}, c.Items())
	c = c.Map(func(i int, _ int) int {
		return i * 2
	})
	assert.Equal(t, []int{2, 4, 6, 8}, c.All())

	// Filter/Where
	c = c.Where(func(v int, _ int) bool {
		return v > 4
	})
	assert.Equal(t, []int{6, 8}, c.Items())

	// Reduce
	assert.Equal(t, 14, c.Reduce(func(a, b int) int {
		return a + b
	}))

	// Each
	var sum int
	c.Each(func(v int, _ int) {
		sum += v
	})
	assert.Equal(t, 14, sum)

	// Find
	v, ok := c.Find(func(v int, _ int) bool {
		return v == 6
	})
	assert.True(t, ok)
	assert.Equal(t, 6, v)
	v2, ok2 := c.Find(func(v int, _ int) bool {
		return v == 10
	})
	assert.False(t, ok2)
	assert.Equal(t, 0, v2)

	// First
	v3, ok3 := c.First()
	assert.True(t, ok3)
	assert.Equal(t, 6, v3)
	v33, ok33 := New([]int{}).First()
	assert.False(t, ok33)
	assert.Equal(t, 0, v33)

	// Index
	v4, ok4 := c.Index(func(i int, _ int) bool {
		return i == 6
	})
	assert.True(t, ok4)
	assert.Equal(t, 0, v4)
	v5, ok5 := c.Index(func(i int, _ int) bool {
		return i == 10
	})
	assert.False(t, ok5)
	assert.Equal(t, -1, v5)

	// Contains/Has/Exists/Missing
	assert.True(t, c.Contains(6))
	assert.True(t, c.Has(6))
	assert.True(t, c.Exists(6))
	assert.False(t, c.Missing(6))
	assert.False(t, c.Contains(10))

	// IsEmpty/NotEmpty
	assert.False(t, c.IsEmpty())
	assert.True(t, c.IsNotEmpty())
	assert.True(t, New([]int{}).IsEmpty())
	assert.False(t, New([]int{}).IsNotEmpty())

	// Last
	v6, ok6 := c.Last()
	assert.True(t, ok6)
	assert.Equal(t, 8, v6)
	v7, ok7 := New([]int{}).Last()
	assert.False(t, ok7)
	assert.Equal(t, 0, v7)

	// Unique
	c = New([]int{1, 2, 3, 2, 1})
	assert.Equal(t, []int{1, 2, 3}, c.Unique().Items())

	// Reverse
	c = New([]int{1, 2, 3})
	assert.Equal(t, []int{3, 2, 1}, c.Reverse().Items())

	// SortBy
	c = New([]int{3, 2, 1})
	assert.Equal(t, []int{1, 2, 3}, c.SortBy(func(a, b int) bool {
		return a < b
	}).Items())

	// Dump
	assert.NotPanics(t, func() {
		c.Dump()
	})
}

func TestCollection_String(t *testing.T) {
	c := New([]string{"a", "b", "c"})
	c.Add("d")
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []string{"a", "b", "c", "d"}, c.Items())
	c = c.Map(func(s string, _ int) string {
		return s + s
	})
	assert.Equal(t, []string{"aa", "bb", "cc", "dd"}, c.Items())

	// Filter/Where
	c = c.Where(func(v string, _ int) bool {
		return v >= "cc"
	})
	assert.Equal(t, []string{"cc", "dd"}, c.Items())

	// Reduce
	assert.Equal(t, "ccdd", c.Reduce(func(a, b string) string {
		return a + b
	}))

	// Each
	var s string
	c.Each(func(v string, _ int) {
		s += v
	})
	assert.Equal(t, "ccdd", s)

	// Find
	v, ok := c.Find(func(v string, _ int) bool {
		return v == "cc"
	})
	assert.True(t, ok)
	assert.Equal(t, "cc", v)
	v2, ok2 := c.Find(func(v string, _ int) bool {
		return v == "ee"
	})
	assert.False(t, ok2)
	assert.Equal(t, "", v2)

	// First
	v3, ok3 := c.First()
	assert.True(t, ok3)
	assert.Equal(t, "cc", v3)
	v33, ok33 := New([]string{}).First()
	assert.False(t, ok33)
	assert.Equal(t, "", v33)

	// Index
	v4, ok4 := c.Index(func(s string, _ int) bool {
		return s == "dd"
	})
	assert.True(t, ok4)
	assert.Equal(t, 1, v4)
	v5, ok5 := c.Index(func(s string, _ int) bool {
		return s == "ee"
	})
	assert.False(t, ok5)
	assert.Equal(t, -1, v5)

	// Contains/Has/Exists/Missing
	assert.True(t, c.Contains("cc"))
	assert.True(t, c.Has("cc"))
	assert.True(t, c.Exists("cc"))
	assert.False(t, c.Missing("cc"))
	assert.False(t, c.Contains("ee"))

	// IsEmpty/NotEmpty
	assert.False(t, c.IsEmpty())
	assert.True(t, c.IsNotEmpty())
	assert.True(t, New([]string{}).IsEmpty())
	assert.False(t, New([]string{}).IsNotEmpty())

	// Last
	v6, ok6 := c.Last()
	assert.True(t, ok6)
	assert.Equal(t, "dd", v6)
	v7, ok7 := New([]string{}).Last()
	assert.False(t, ok7)
	assert.Equal(t, "", v7)

	// Unique
	c = New([]string{"a", "b", "c", "b", "a"})
	assert.Equal(t, []string{"a", "b", "c"}, c.Unique().Items())

	// Reverse
	c = New([]string{"a", "b", "c"})
	assert.Equal(t, []string{"c", "b", "a"}, c.Reverse().Items())

	// SortBy
	c = New([]string{"c", "b", "a"})
	assert.Equal(t, []string{"a", "b", "c"}, c.SortBy(func(a, b string) bool {
		return a < b
	}).Items())

	// Dump
	assert.NotPanics(t, func() {
		c.Dump()
	})
}

type User struct {
	Name string
}

func TestCollection_Ptr(t *testing.T) {
	c := New([]*User{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	c.Add(&User{Name: "d"})
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []*User{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}}, c.Items())
	c = c.Map(func(u *User, _ int) *User {
		return &User{Name: u.Name + u.Name}
	})
	assert.Equal(t, []*User{{Name: "aa"}, {Name: "bb"}, {Name: "cc"}, {Name: "dd"}}, c.Items())

	// Filter/Where
	c = c.Where(func(u *User, _ int) bool {
		return u.Name >= "cc"
	})
	assert.Equal(t, []*User{{Name: "cc"}, {Name: "dd"}}, c.Items())

	// Reduce
	assert.Equal(t, "ccdd", c.Reduce(func(a, b *User) *User {
		if a == nil {
			return b
		}
		return &User{Name: a.Name + b.Name}
	}).Name)

	// Each
	var s string
	c.Each(func(u *User, _ int) {
		s += u.Name
	})
	assert.Equal(t, "ccdd", s)

	// Find
	v, ok := c.Find(func(u *User, _ int) bool {
		return u.Name == "cc"
	})
	assert.True(t, ok)
	assert.Equal(t, "cc", v.Name)
	v2, ok2 := c.Find(func(u *User, _ int) bool {
		return u.Name == "ee"
	})
	assert.False(t, ok2)
	assert.Nil(t, v2)

	// First
	v3, ok3 := c.First()
	assert.True(t, ok3)
	assert.Equal(t, "cc", v3.Name)
	v33, ok33 := New([]*User{}).First()
	assert.False(t, ok33)
	assert.Nil(t, v33)

	// Index
	v4, ok4 := c.Index(func(u *User, _ int) bool {
		return u.Name == "dd"
	})
	assert.True(t, ok4)
	assert.Equal(t, 1, v4)
	v5, ok5 := c.Index(func(u *User, _ int) bool {
		return u.Name == "ee"
	})
	assert.False(t, ok5)
	assert.Equal(t, -1, v5)

	// Contains/Has/Exists/Missing
	c2 := New([]User{{Name: "cc"}, {Name: "dd"}})
	assert.True(t, c2.Contains(User{Name: "cc"}))
	assert.True(t, c2.Has(User{Name: "cc"}))
	assert.True(t, c2.Exists(User{Name: "cc"}))
	assert.False(t, c2.Missing(User{Name: "cc"}))
	assert.False(t, c2.Contains(User{Name: "ee"}))

	// IsEmpty/NotEmpty
	assert.False(t, c.IsEmpty())
	assert.True(t, c.IsNotEmpty())
	assert.True(t, New([]*User{}).IsEmpty())
	assert.False(t, New([]*User{}).IsNotEmpty())

	// Last
	v6, ok6 := c.Last()
	assert.True(t, ok6)
	assert.Equal(t, "dd", v6.Name)
	v7, ok7 := New([]*User{}).Last()
	assert.False(t, ok7)
	assert.Nil(t, v7)

	// Unique
	c3 := New([]User{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "b"}, {Name: "a"}})
	assert.Equal(t, []User{{Name: "a"}, {Name: "b"}, {Name: "c"}}, c3.Unique().Items())

	// Reverse
	c4 := New([]User{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	assert.Equal(t, []User{{Name: "c"}, {Name: "b"}, {Name: "a"}}, c4.Reverse().Items())

	// SortBy
	c5 := New([]User{{Name: "c"}, {Name: "b"}, {Name: "a"}})
	assert.Equal(t, []User{{Name: "a"}, {Name: "b"}, {Name: "c"}}, c5.SortBy(func(a, b User) bool {
		return a.Name < b.Name
	}).Items())

	// Dump
	assert.NotPanics(t, func() {
		c.Dump()
	})
}

func TestCollection_When(t *testing.T) {
	c := New([]int{1, 2, 3})
	c.When(true, func(c *Collection[int]) {
		c.Add(4)
	})
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []int{1, 2, 3, 4}, c.Items())
	c.When(false, func(c *Collection[int]) {
		c.Add(5)
	})
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []int{1, 2, 3, 4}, c.Items())
}

func TestCollection_Unless(t *testing.T) {
	c := New([]int{1, 2, 3})
	c.Unless(false, func(c *Collection[int]) {
		c.Add(4)
	})
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []int{1, 2, 3, 4}, c.Items())
	c.Unless(true, func(c *Collection[int]) {
		c.Add(5)
	})
	assert.Equal(t, 4, c.Len())
	assert.Equal(t, []int{1, 2, 3, 4}, c.Items())
}
