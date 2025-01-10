package coordinator

import (
	"sync"
	"testing"
	"time"
)

func TestCoordinator(t *testing.T) {
	c := NewCoordinator()
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-c.Done():
				return
			case <-timer.C:
				t.Error("timeout")
				return
			}
		}
	}()

	c.Close()
	wg.Wait()
}

func TestCoordinator2(t *testing.T) {
	c := NewCoordinator()
	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-c.Done():
				t.Error("timeout")
				return
			case <-timer.C:
				return
			}
		}
	}()

	time.Sleep(2 * time.Second)

	c.Close()
	wg.Wait()
}
