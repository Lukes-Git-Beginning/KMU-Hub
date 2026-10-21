package formulare

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// panicClaimRepo panics on the first ClaimPendingDeliveries call and counts all calls.
type panicClaimRepo struct {
	*stubRepo
	calls atomic.Int32
}

func (p *panicClaimRepo) ClaimPendingDeliveries(ctx context.Context, batchSize int) ([]*WebhookDelivery, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in batch")
	}
	return p.stubRepo.ClaimPendingDeliveries(ctx, batchSize)
}

// A panic in the first batch must not end Run: later ticks still claim.
func TestWorker_Run_SurvivesPanicInBatch(t *testing.T) {
	repo := &panicClaimRepo{stubRepo: newStubRepo()}
	w := NewWebhookWorker(repo, nil)
	w.pollInterval = 5 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	deadline := time.After(2 * time.Second)
	for repo.calls.Load() < 3 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("worker stopped after a panic: %d claim calls", repo.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
