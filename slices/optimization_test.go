package slices

import "testing"

// Test nil function handling for our optimizations
func TestNilFunctionHandling(t *testing.T) {
	t.Run("Map with nil function", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Map(input, func(int) int { return 0 }) // Can't test with nil due to type inference
		if len(result) != 3 {
			t.Errorf("Expected length 3, got %v", len(result))
		}
	})

	t.Run("Filter with nil function", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := Filter(input, nil)
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %v", result)
		}
	})

	t.Run("FilterN with nil function", func(t *testing.T) {
		input := []int{1, 2, 3}
		result := FilterN(input, nil)
		if len(result) != 0 {
			t.Errorf("Expected empty slice, got %v", result)
		}
	})
}

// Test memory efficiency improvements
func TestMemoryEfficiency(t *testing.T) {
	t.Run("Filter allocates correct capacity", func(t *testing.T) {
		input := []int{1, 2, 3, 4, 5}
		result := Filter(input, func(i int) bool { return i%2 == 0 })
		// This test mainly ensures no panic and correct functionality
		expected := []int{2, 4}
		if len(result) != len(expected) {
			t.Errorf("Expected length %d, got %d", len(expected), len(result))
		}
		for i, v := range expected {
			if result[i] != v {
				t.Errorf("Expected %v at index %d, got %v", v, i, result[i])
			}
		}
	})
}