package main

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestMeasureResourceUsageTracksElapsedAndGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()

	usage := measureResourceUsage(func() {
		var wg sync.WaitGroup
		wg.Add(4)

		for i := 0; i < 4; i++ {
			go func() {
				defer wg.Done()
				time.Sleep(15 * time.Millisecond)
			}()
		}

		wg.Wait()
	})

	if usage.ElapsedMS <= 0 {
		t.Errorf("ElapsedMS = %v, se esperaba > 0", usage.ElapsedMS)
	}

	if usage.PeakGoroutines < before {
		t.Errorf(
			"PeakGoroutines = %d, se esperaba >= goroutines previas (%d)",
			usage.PeakGoroutines,
			before,
		)
	}
}
