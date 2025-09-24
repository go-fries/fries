package slices

import (
	"testing"
)

// Benchmark to demonstrate the memory allocation improvements
func BenchmarkFilterMemoryOptimization(b *testing.B) {
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}
	
	b.Run("OptimizedFilter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := Filter(input, func(x int) bool { return x%2 == 0 })
			_ = result
		}
	})
}

func BenchmarkFilterVsFilterN(b *testing.B) {
	input := make([]int, 100)
	for i := range input {
		input[i] = i
	}
	
	b.Run("Filter", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := Filter(input, func(x int) bool { return x%2 == 0 })
			_ = result
		}
	})
	
	b.Run("FilterN", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			result := FilterN(input, func(x int, _ int) bool { return x%2 == 0 })
			_ = result
		}
	})
}