package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// panicOnceRepo panics on the first ListDueSchedules call and counts all calls.
type panicOnceRepo struct {
	*mockRepo
	calls atomic.Int32
}

func (p *panicOnceRepo) ListDueSchedules(ctx context.Context, now time.Time) ([]*berichte.Schedule, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in tick")
	}
	return p.mockRepo.ListDueSchedules(ctx, now)
}

// A panic in the first tick (the immediate one in Run) must not end the loop:
// the following ticks still run.
func TestScheduler_Run_SurvivesPanicInTick(t *testing.T) {
	repo := &panicOnceRepo{mockRepo: newMockRepo()}
	s := New(repo, &mockRunner{}, &mockExporter{}, &mockMailer{},
		Config{Clock: &fixedClock{t: time.Now()}, Tick: 5 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("scheduler stopped ticking after a panic: %d ListDueSchedules calls", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
