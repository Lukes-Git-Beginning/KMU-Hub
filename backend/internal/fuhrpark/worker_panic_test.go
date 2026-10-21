package fuhrpark

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// A panic in the first scan must not end Run: later run ticks still scan.
func TestTuevWorker_Run_SurvivesPanicInScan(t *testing.T) {
	w := NewTuevWorker(newMockRepository(), nil, nil)
	w.runInterval = 5 * time.Millisecond

	var calls atomic.Int32
	scan := func(context.Context) error {
		if calls.Add(1) == 1 {
			panic("boom in scan")
		}
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx, scan) }()

	deadline := time.After(2 * time.Second)
	for calls.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("worker stopped after a panic: %d scans", calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
