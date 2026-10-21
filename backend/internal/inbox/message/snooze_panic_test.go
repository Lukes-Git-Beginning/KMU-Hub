package message

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// panicUnsnoozeRepo panics on the first UnsnoozeExpired call and counts all calls.
type panicUnsnoozeRepo struct {
	*mockRepository
	calls atomic.Int32
}

func (p *panicUnsnoozeRepo) UnsnoozeExpired(ctx context.Context) (int, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in snooze tick")
	}
	return p.mockRepository.UnsnoozeExpired(ctx)
}

// A panic in the first tick must not end the snooze worker: later ticks still run.
func TestStartSnoozeWorker_SurvivesPanicInTick(t *testing.T) {
	repo := &panicUnsnoozeRepo{mockRepository: newMockRepository()}
	svc := NewService(repo, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.StartSnoozeWorker(ctx, 5*time.Millisecond)

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			t.Fatalf("snooze worker stopped after a panic: %d ticks", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
}
