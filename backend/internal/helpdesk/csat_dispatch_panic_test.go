package helpdesk

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

)

// panicListRepo panics on the first ListDueCsatSurveys call and counts all calls.
type panicListRepo struct {
	*fakeDispatchRepo
	calls atomic.Int32
}

func (p *panicListRepo) ListDueCsatSurveys(ctx context.Context, now time.Time, maxAttempts int16, limit int) ([]*DueCsatSurvey, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in tick")
	}
	return p.fakeDispatchRepo.ListDueCsatSurveys(ctx, now, maxAttempts, limit)
}

// A panic in the immediate first tick must not end Run: later ticks still poll.
func TestDispatcher_Run_SurvivesPanicInTick(t *testing.T) {
	t.Parallel()

	repo := &panicListRepo{fakeDispatchRepo: newFakeDispatchRepo()}
	d := quietDispatcher(repo, &fakeMailer{}, &fakeConfigLoader{cfg: CsatConfig{Enabled: true}})
	d.tick = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { d.Run(ctx); close(done) }()

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("dispatcher stopped after a panic: %d list calls", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-done
}
