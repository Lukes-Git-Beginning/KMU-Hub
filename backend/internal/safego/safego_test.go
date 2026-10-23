package safego

import (
	"sync"
	"testing"
)

func TestDo_RecoversPanicAndReportsIt(t *testing.T) {
	if Do("boom", func() { panic("boom") }) {
		t.Fatal("Do reported success for a panicking fn")
	}
	if !Do("fine", func() {}) {
		t.Fatal("Do reported failure for a clean fn")
	}
}

// A panicking tick must not end the loop: the second tick still runs.
func TestDo_NextTickStillRuns(t *testing.T) {
	runs := 0
	tick := func() {
		runs++
		if runs == 1 {
			panic("first tick")
		}
	}
	Do("tick", tick)
	Do("tick", tick)
	if runs != 2 {
		t.Fatalf("runs = %d, want 2", runs)
	}
}

func TestGo_PanicDoesNotCrashProcess(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	Go("async", func() {
		defer wg.Done()
		panic("async boom")
	})
	wg.Wait()
}
