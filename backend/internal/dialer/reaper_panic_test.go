package dialer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// panicStaleRepo panics on the first ListStaleInProgressContacts call and counts all calls.
type panicStaleRepo struct {
	*mockCampaignRepo
	calls atomic.Int32
}

func (p *panicStaleRepo) ListStaleInProgressContacts(ctx context.Context, cutoff time.Time, limit int) ([]StaleContact, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in sweep")
	}
	return p.mockCampaignRepo.ListStaleInProgressContacts(ctx, cutoff, limit)
}

// A panic in the first sweep must not end the reaper: later ticks still sweep.
func TestRunStaleContactReaper_SurvivesPanicInSweep(t *testing.T) {
	repo := &panicStaleRepo{mockCampaignRepo: &mockCampaignRepo{}}
	svc := NewService(repo, nil, nil, nil, nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { RunStaleContactReaper(ctx, svc, time.Hour, 5*time.Millisecond); close(done) }()

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("reaper stopped after a panic: %d sweeps", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-done
}
