package formulare

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
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


// panicDeliverRepo panics while loading the webhook of one poisoned delivery.
type panicDeliverRepo struct {
	*stubRepo
	poison uuid.UUID
}

func (p *panicDeliverRepo) GetWebhook(ctx context.Context, id, tenantID uuid.UUID) (*FormWebhook, error) {
	if id == p.poison {
		panic("boom in delivery")
	}
	return p.stubRepo.GetWebhook(ctx, id, tenantID)
}

// A panic in one delivery must not strand the rest of the batch, and the
// poisoned row must be recorded as a failed attempt so it eventually dies.
func TestWorker_ProcessBatch_PanicInOneDeliveryKeepsTheRest(t *testing.T) {
	tenantID := uuid.New()
	poison := uuid.New()
	repo := &panicDeliverRepo{stubRepo: newStubRepo(), poison: poison}
	healthy := &FormWebhook{ID: uuid.New(), TenantID: tenantID, Active: false}
	repo.webhooks[healthy.ID] = healthy
	bad := &WebhookDelivery{ID: uuid.New(), TenantID: tenantID, WebhookID: poison,
		Status: WebhookDeliveryStatusPending, MaxAttempts: 5}
	good := &WebhookDelivery{ID: uuid.New(), TenantID: tenantID, WebhookID: healthy.ID,
		Status: WebhookDeliveryStatusPending, MaxAttempts: 5}
	repo.deliveries[bad.ID] = bad
	repo.deliveries[good.ID] = good

	w := NewWebhookWorker(repo, nil)
	if err := w.processBatch(context.Background()); err != nil {
		t.Fatalf("processBatch: %v", err)
	}

	if bad.Status != WebhookDeliveryStatusFailed || bad.AttemptCount != 1 {
		t.Errorf("poisoned delivery: status %q, attempts %d; want failed, 1", bad.Status, bad.AttemptCount)
	}
	if good.Status != WebhookDeliveryStatusDead {
		t.Errorf("healthy delivery not processed: status %q", good.Status)
	}
}
