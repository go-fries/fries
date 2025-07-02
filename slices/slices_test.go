package slices

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type T struct {
	A string
}

func TestMap(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	s2 := []int{1, 2, 3, 4, 5}
	s3 := []string{"1", "2", "3", "4", "5"}

	t.Run("Map", func(t *testing.T) {
		assert.Equal(t, []int{2, 4, 6, 8, 10}, Map(s1, func(n int) int { return n * 2 }))
		assert.Equal(t, []string{"1", "2", "3", "4", "5"}, Map(s2, func(n int) string { return strconv.Itoa(n) }))
		assert.Equal(t, []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}, Map(s3, func(s string) T { return T{s} }))
	})

	t.Run("MapN", func(t *testing.T) {
		assert.Equal(t, []int{2, 4, 6, 8, 10}, MapN(s1, func(n, i int) int {
			if n == 3 {
				assert.Equal(t, 2, i)
			}
			return n * 2
		}))
		assert.Equal(t, []string{"1", "2", "3", "4", "5"}, MapN(s2, func(n, _ int) string { return strconv.Itoa(n) }))
		assert.Equal(t, []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}, MapN(s3, func(s string, _ int) T { return T{s} }))
	})
}

func TestKeyMap(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, map[int]int{1: 2, 2: 4, 3: 6, 4: 8, 5: 10}, KeyMap(s1, func(n, _ int) (int, int) {
		return n, n * 2
	}))

	s2 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, map[int]string{1: "1", 2: "2", 3: "3", 4: "4", 5: "5"}, KeyMap(s2, func(n, _ int) (int, string) {
		return n, strconv.Itoa(n)
	}))

	s3 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, map[string]T{"1": {"1"}, "2": {"2"}, "3": {"3"}, "4": {"4"}, "5": {"5"}}, KeyMap(s3, func(s string, _ int) (string, T) { //nolint:lll
		return s, T{s}
	}))

	s4 := []string{"1", "2", "3", "3", "5"}
	assert.Equal(t, map[string]int{"1": 1, "2": 2, "3": 4, "5": 5}, KeyMap(s4, func(s string, i int) (string, int) {
		if i == 2 {
			assert.Equal(t, "3", s)
		}
		return s, i + 1
	}))
}

func TestEach(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	var result []int
	Each(s1, func(n, i int) {
		if n == 3 {
			assert.Equal(t, 2, i)
		}
		result = append(result, n*2)
	})
	assert.Equal(t, []int{2, 4, 6, 8, 10}, result)

	s2 := []int{1, 2, 3, 4, 5}
	var result2 []string
	Each(s2, func(n, _ int) { result2 = append(result2, strconv.Itoa(n)) })
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, result2)

	s3 := []string{"1", "2", "3", "4", "5"}
	var result3 []T
	Each(s3, func(s string, _ int) { result3 = append(result3, T{s}) })
}

func TestPrepend(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, []int{0, 1, 2, 3}, Prepend(s1, 0))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, []string{"0", "1", "2", "3"}, Prepend(s2, "0"))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.Equal(t, []T{{"0"}, {"1"}, {"2"}, {"3"}}, Prepend(s3, T{"0"}))
}

func TestAppend(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, []int{1, 2, 3, 4}, Append(s1, 4))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, []string{"1", "2", "3", "4"}, Append(s2, "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}, {"4"}}, Append(s3, T{"4"}))
}

func TestFilter(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, []int{2, 4}, Filter(s1, func(n, _ int) bool { return n%2 == 0 }))

	s2 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, []int{1, 3, 5}, Filter(s2, func(n, _ int) bool { return n%2 != 0 }))

	s3 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, []string{"1", "2", "3"}, Filter(s3, func(s string, _ int) bool { return s < "4" }))

	s4 := []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}}, Filter(s4, func(t T, _ int) bool { return t.A < "4" }))

	s5 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, []string{"1", "2", "3"}, Filter(s5, func(_ string, i int) bool { return i <= 2 }))
}

func TestReduce(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, 15, Reduce(s1, func(acc, n, _ int) int { return acc + n }))

	s2 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, 0, Reduce(s2, func(acc, n, _ int) int { return acc * n }))
	assert.Equal(t, 120, Reduce(s2, func(acc, n, _ int) int { return acc * n }, 1))

	s3 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, "12345", Reduce(s3, func(acc, s string, _ int) string { return acc + s }))
}

func TestReverse(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, []int{5, 4, 3, 2, 1}, Reverse(s1))

	s2 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, []string{"5", "4", "3", "2", "1"}, Reverse(s2))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	assert.Equal(t, []T{{"5"}, {"4"}, {"3"}, {"2"}, {"1"}}, Reverse(s3))
}

func TestConcat(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{4, 5, 6}
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, Concat(s1, s2))

	s3 := []string{"1", "2", "3"}
	s4 := []string{"4", "5", "6"}
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, Concat(s3, s4))

	s5 := []T{{"1"}, {"2"}, {"3"}}
	s6 := []T{{"4"}, {"5"}, {"6"}}
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}, {"6"}}, Concat(s5, s6))
}

func TestIsEmptyAndIsNotEmpty(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.False(t, IsEmpty(s1))
	assert.True(t, IsNotEmpty(s1))

	var s2 []int
	assert.True(t, IsEmpty(s2))
	assert.False(t, IsNotEmpty(s2))

	s3 := []string{"1", "2", "3"}
	assert.False(t, IsEmpty(s3))
	assert.True(t, IsNotEmpty(s3))

	var s4 []string
	assert.True(t, IsEmpty(s4))
	assert.False(t, IsNotEmpty(s4))

	s5 := []T{{"1"}, {"2"}, {"3"}}
	assert.False(t, IsEmpty(s5))
	assert.True(t, IsNotEmpty(s5))

	var s6 []T
	assert.True(t, IsEmpty(s6))
	assert.False(t, IsNotEmpty(s6))
}

func TestContains(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.True(t, Contains(s1, 1))
	assert.False(t, Contains(s1, 4))

	s2 := []string{"1", "2", "3"}
	assert.True(t, Contains(s2, "1"))
	assert.False(t, Contains(s2, "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.True(t, Contains(s3, T{"1"}))
	assert.False(t, Contains(s3, T{"4"}))
}

func TestIndexOf(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, 0, IndexOf(s1, 1))
	assert.Equal(t, -1, IndexOf(s1, 4))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, 0, IndexOf(s2, "1"))
	assert.Equal(t, -1, IndexOf(s2, "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.Equal(t, 0, IndexOf(s3, T{"1"}))
	assert.Equal(t, -1, IndexOf(s3, T{"4"}))
}

func TestLastIndexOf(t *testing.T) {
	s1 := []int{1, 2, 3, 1}
	assert.Equal(t, 3, LastIndexOf(s1, 1))
	assert.Equal(t, -1, LastIndexOf(s1, 4))

	s2 := []string{"1", "2", "3", "1"}
	assert.Equal(t, 3, LastIndexOf(s2, "1"))
	assert.Equal(t, -1, LastIndexOf(s2, "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"1"}}
	assert.Equal(t, 3, LastIndexOf(s3, T{"1"}))
	assert.Equal(t, -1, LastIndexOf(s3, T{"4"}))
}

func TestUnique(t *testing.T) {
	s1 := []int{1, 2, 3, 1, 2, 3}
	assert.Equal(t, []int{1, 2, 3}, Unique(s1))

	s2 := []string{"1", "2", "3", "1", "2", "3"}
	assert.Equal(t, []string{"1", "2", "3"}, Unique(s2))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"1"}, {"2"}, {"3"}}
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}}, Unique(s3))
}

func TestUniqueBy(t *testing.T) {
	s1 := []int{1, 2, 3, 1, 2, 3}
	assert.Equal(t, []int{1, 2, 3}, UniqueBy(s1, func(n, _ int) int { return n }))

	s2 := []string{"1", "2", "3", "1", "2", "3"}
	assert.Equal(t, []string{"1", "2", "3"}, UniqueBy(s2, func(s string, _ int) string { return s }))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"1"}, {"2"}, {"3"}}
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}}, UniqueBy(s3, func(t T, _ int) string { return t.A }))

	s4 := []string{"apple", "apple2", "cherry"}
	assert.Equal(t, []string{"apple", "cherry"}, UniqueBy(s4, func(s string, _ int) string { return s[:1] }))
}

func TestDifference(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	assert.Equal(t, []int{1}, Difference(s1, s2))

	s3 := []string{"1", "2", "3"}
	s4 := []string{"2", "3", "4"}
	assert.Equal(t, []string{"1"}, Difference(s3, s4))

	s5 := []T{{"1"}, {"2"}, {"3"}}
	s6 := []T{{"2"}, {"3"}, {"4"}}
	assert.Equal(t, []T{{"1"}}, Difference(s5, s6))
}

func TestIntersect(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	assert.Equal(t, []int{2, 3}, Intersect(s1, s2))

	s3 := []string{"1", "2", "3"}
	s4 := []string{"2", "3", "4"}
	assert.Equal(t, []string{"2", "3"}, Intersect(s3, s4))

	s5 := []T{{"1"}, {"2"}, {"3"}}
	s6 := []T{{"2"}, {"3"}, {"4"}}
	assert.Equal(t, []T{{"2"}, {"3"}}, Intersect(s5, s6))
}

func TestOnly(t *testing.T) {
	s1 := []int{1, 2, 3, 4}
	assert.Equal(t, []int{2, 4}, Only(s1, 2, 4))

	s2 := []string{"1", "2", "3", "4"}
	assert.Equal(t, []string{"2", "4"}, Only(s2, "2", "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}}
	assert.Equal(t, []T{{"2"}, {"4"}}, Only(s3, T{"2"}, T{"4"}))
}

func TestWithoutAndRemove(t *testing.T) {
	s1 := []int{1, 2, 3, 4}
	assert.Equal(t, []int{1, 3}, Without(s1, 2, 4))
	assert.Equal(t, []int{1, 3}, Remove(s1, 2, 4))

	s2 := []string{"1", "2", "3", "4"}
	assert.Equal(t, []string{"1", "3"}, Without(s2, "2", "4"))
	assert.Equal(t, []string{"1", "3"}, Remove(s2, "2", "4"))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}}
	assert.Equal(t, []T{{"1"}, {"3"}, {"4"}}, Without(s3, T{"2"}))
	assert.Equal(t, []T{{"1"}, {"3"}, {"4"}}, Remove(s3, T{"2"}))
}

func TestPartition(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	odd, even := Partition(s1, func(n, _ int) bool { return n%2 != 0 })
	assert.Equal(t, []int{1, 3, 5}, odd)
	assert.Equal(t, []int{2, 4}, even)

	s2 := []string{"1", "2", "3", "4", "5"}
	lessThan3, greaterThan3 := Partition(s2, func(s string, _ int) bool { return s < "3" })
	assert.Equal(t, []string{"1", "2"}, lessThan3)
	assert.Equal(t, []string{"3", "4", "5"}, greaterThan3)

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	lessThan4, greaterThan4 := Partition(s3, func(t T, _ int) bool { return t.A < "4" })
	assert.Equal(t, []T{{"1"}, {"2"}, {"3"}}, lessThan4)
	assert.Equal(t, []T{{"4"}, {"5"}}, greaterThan4)
}

func TestChunk(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, Chunk(s1, 2))

	s2 := []string{"1", "2", "3", "4", "5"}
	assert.Equal(t, [][]string{{"1", "2"}, {"3", "4"}, {"5"}}, Chunk(s2, 2))

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	assert.Equal(t, [][]T{{{"1"}, {"2"}}, {{"3"}, {"4"}}, {{"5"}}}, Chunk(s3, 2))
}

func TestGroupBy(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5}
	result1 := GroupBy(s1, func(n, _ int) string { return strconv.Itoa(n % 2) })
	assert.Equal(t, map[string][]int{"0": {2, 4}, "1": {1, 3, 5}}, result1)

	s2 := []string{"1", "2", "3", "4", "5"}
	result2 := GroupBy(s2, func(s string, _ int) string { return s })
	assert.Equal(t, map[string][]string{"1": {"1"}, "2": {"2"}, "3": {"3"}, "4": {"4"}, "5": {"5"}}, result2)

	s3 := []T{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	result3 := GroupBy(s3, func(t T, _ int) string { return t.A })
	assert.Equal(t, map[string][]T{"1": {{"1"}}, "2": {{"2"}}, "3": {{"3"}}, "4": {{"4"}}, "5": {{"5"}}}, result3)
}

func TestFirst(t *testing.T) {
	s1 := []int{1, 2, 3}
	first1, ok1 := First(s1)
	assert.Equal(t, 1, first1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	first2, ok2 := First(s2)
	assert.Equal(t, "1", first2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	first3, ok3 := First(s3)
	assert.Equal(t, T{"1"}, first3)
	assert.True(t, ok3)

	s4 := []int{}
	first4, ok4 := First(s4)
	assert.Equal(t, 0, first4)
	assert.False(t, ok4)

	s5 := []string{}
	first5, ok5 := First(s5)
	assert.Equal(t, "", first5)
	assert.False(t, ok5)

	var s6 []*T
	first6, ok6 := First(s6)
	assert.Equal(t, (*T)(nil), first6)
	assert.False(t, ok6)
}

func TestLast(t *testing.T) {
	s1 := []int{1, 2, 3}
	last1, ok1 := Last(s1)
	assert.Equal(t, 3, last1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	last2, ok2 := Last(s2)
	assert.Equal(t, "3", last2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	last3, ok3 := Last(s3)
	assert.Equal(t, T{"3"}, last3)
	assert.True(t, ok3)

	s4 := []int{}
	last4, ok4 := Last(s4)
	assert.Equal(t, 0, last4)
	assert.False(t, ok4)

	s5 := []string{}
	last5, ok5 := Last(s5)
	assert.Equal(t, "", last5)
	assert.False(t, ok5)

	var s6 []*T
	last6, ok6 := Last(s6)
	assert.Equal(t, (*T)(nil), last6)
	assert.False(t, ok6)
}

func TestFind(t *testing.T) {
	s1 := []int{1, 2, 3}
	find1, ok1 := Find(s1, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 2, find1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	find2, ok2 := Find(s2, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, "2", find2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	find3, ok3 := Find(s3, func(t T, _ int) bool { return t.A == "2" })
	assert.Equal(t, T{"2"}, find3)
	assert.True(t, ok3)

	s4 := []int{}
	find4, ok4 := Find(s4, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 0, find4)
	assert.False(t, ok4)

	s5 := []string{}
	find5, ok5 := Find(s5, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, "", find5)
	assert.False(t, ok5)

	var s6 []*T
	find6, ok6 := Find(s6, func(t *T, _ int) bool { return t.A == "2" })
	assert.Equal(t, (*T)(nil), find6)
	assert.False(t, ok6)
}

func TestFindLast(t *testing.T) {
	s1 := []int{1, 2, 3}
	find1, ok1 := FindLast(s1, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 2, find1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	find2, ok2 := FindLast(s2, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, "2", find2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	find3, ok3 := FindLast(s3, func(t T, _ int) bool { return t.A == "2" })
	assert.Equal(t, T{"2"}, find3)
	assert.True(t, ok3)

	s4 := []int{}
	find4, ok4 := FindLast(s4, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 0, find4)
	assert.False(t, ok4)

	s5 := []string{}
	find5, ok5 := FindLast(s5, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, "", find5)
	assert.False(t, ok5)

	var s6 []*T
	find6, ok6 := FindLast(s6, func(t *T, _ int) bool { return t.A == "2" })
	assert.Equal(t, (*T)(nil), find6)
	assert.False(t, ok6)
}

func TestIndex(t *testing.T) {
	s1 := []int{1, 2, 3}
	index1, ok1 := Index(s1, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 1, index1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	index2, ok2 := Index(s2, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, 1, index2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	index3, ok3 := Index(s3, func(t T, _ int) bool { return t.A == "2" })
	assert.Equal(t, 1, index3)
	assert.True(t, ok3)

	s4 := []int{}
	index4, ok4 := Index(s4, func(n int, _ int) bool { return n%2 == 0 })
	assert.Equal(t, -1, index4)
	assert.False(t, ok4)

	s5 := []string{}
	index5, ok5 := Index(s5, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, -1, index5)
	assert.False(t, ok5)

	var s6 []*T
	index6, ok6 := Index(s6, func(t *T, _ int) bool { return t.A == "2" })
	assert.Equal(t, -1, index6)
	assert.False(t, ok6)
}

func TestLastIndex(t *testing.T) {
	s1 := []int{1, 2, 3}
	index1, ok1 := LastIndex(s1, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, 1, index1)
	assert.True(t, ok1)

	s2 := []string{"1", "2", "3"}
	index2, ok2 := LastIndex(s2, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, 1, index2)
	assert.True(t, ok2)

	s3 := []T{{"1"}, {"2"}, {"3"}}
	index3, ok3 := LastIndex(s3, func(t T, _ int) bool { return t.A == "2" })
	assert.Equal(t, 1, index3)
	assert.True(t, ok3)

	s4 := []int{}
	index4, ok4 := LastIndex(s4, func(n, _ int) bool { return n%2 == 0 })
	assert.Equal(t, -1, index4)
	assert.False(t, ok4)

	s5 := []string{}
	index5, ok5 := LastIndex(s5, func(s string, _ int) bool { return s == "2" })
	assert.Equal(t, -1, index5)
	assert.False(t, ok5)

	var s6 []*T
	index6, ok6 := LastIndex(s6, func(t *T, _ int) bool { return t.A == "2" })
	assert.Equal(t, -1, index6)
	assert.False(t, ok6)
}

func TestFill(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, []int{1, 1, 1}, Fill(s1, 1))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, []string{"1", "1", "1"}, Fill(s2, "1"))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.Equal(t, []T{{"1"}, {"1"}, {"1"}}, Fill(s3, T{"1"}))

	s4 := make([]int, 3)
	assert.Equal(t, []int{1, 1, 1}, Fill(s4, 1))
}

func TestRandom(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Contains(t, s1, Random(s1))
	assert.True(t, func() bool {
		for {
			v1, v2 := Random(s1), Random(s1)
			if v1 != v2 {
				return true
			}
		}
	}())

	s2 := []string{"1", "2", "3"}
	assert.Contains(t, s2, Random(s2))

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.Contains(t, s3, Random(s3))
}

func TestShuffle(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.True(t, func() bool {
		for {
			shuffled := Shuffle(s1)
			if s1[0] != shuffled[0] || s1[1] != shuffled[1] || s1[2] != shuffled[2] {
				return true
			}
		}
	}())

	s2 := []string{"1", "2", "3"}
	assert.True(t, func() bool {
		for {
			shuffled := Shuffle(s2)
			if s2[0] != shuffled[0] || s2[1] != shuffled[1] || s2[2] != shuffled[2] {
				return true
			}
		}
	}())

	s3 := []T{{"1"}, {"2"}, {"3"}}
	assert.True(t, func() bool {
		for {
			shuffled := Shuffle(s3)
			if s3[0] != shuffled[0] || s3[1] != shuffled[1] || s3[2] != shuffled[2] {
				return true
			}
		}
	}())
}

func TestMin(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, 1, Min(s1))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, "1", Min(s2))

	s3 := []int{3, 2, 1}
	assert.Equal(t, 1, Min(s3))
}

func TestMax(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, 3, Max(s1))

	s2 := []string{"1", "2", "3"}
	assert.Equal(t, "3", Max(s2))
}

func TestSum(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, 6, Sum(s1))

	s2 := []int{}
	assert.Equal(t, 0, Sum(s2))
}

func TestLength(t *testing.T) {
	s1 := []int{1, 2, 3}
	assert.Equal(t, 3, Length(s1))

	s2 := []int{}
	assert.Equal(t, 0, Length(s2))
}
