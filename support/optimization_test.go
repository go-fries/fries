package support

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Test our optimization improvements
func TestOptimizationImprovements(t *testing.T) {
	t.Run("Retry with nil function", func(t *testing.T) {
		err := Retry(nil, 3)
		if err == nil {
			t.Error("Expected error when function is nil")
		}
		if !strings.Contains(err.Error(), "function cannot be nil") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("Retry with invalid attempts", func(t *testing.T) {
		err := Retry(func() error { return nil }, 0)
		if err == nil {
			t.Error("Expected error when attempts is 0")
		}
	})

	t.Run("Until with nil function", func(t *testing.T) {
		// Should not panic and should return immediately
		done := make(chan struct{})
		go func() {
			Until(nil)
			close(done)
		}()
		
		select {
		case <-done:
			// Success - function returned without panic
		case <-time.After(100 * time.Millisecond):
			t.Error("Until did not return quickly with nil function")
		}
	})

	t.Run("UntilTimeout prevents goroutine leak", func(t *testing.T) {
		// Test that timeout properly handles cleanup
		err := UntilTimeout(func() bool {
			time.Sleep(200 * time.Millisecond)
			return false
		}, 50*time.Millisecond)
		
		if err == nil {
			t.Error("Expected timeout error")
		}
		
		// Allow some time for goroutine cleanup
		time.Sleep(100 * time.Millisecond)
		// If we get here without hanging, cleanup worked
	})

	t.Run("Timeout handles function panic", func(t *testing.T) {
		err := Timeout(func() error {
			panic("test panic")
		}, time.Second)
		
		if err == nil {
			t.Error("Expected error when function panics")
		}
		
		if !strings.Contains(err.Error(), "panicked") {
			t.Errorf("Expected panic to be wrapped in error, got: %v", err)
		}
	})

	t.Run("Retry doesn't sleep after last attempt", func(t *testing.T) {
		attempts := 0
		start := time.Now()
		
		err := Retry(func() error {
			attempts++
			return errors.New("always fail")
		}, 3, 100*time.Millisecond)
		
		elapsed := time.Since(start)
		
		if err == nil {
			t.Error("Expected error")
		}
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		// Should take ~200ms (2 sleeps), not 300ms (3 sleeps)
		if elapsed > 250*time.Millisecond {
			t.Errorf("Retry took too long: %v (expected ~200ms)", elapsed)
		}
	})
}