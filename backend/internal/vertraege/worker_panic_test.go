package vertraege

import (
	"context"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// panicRepo panics on the first call of each worker query and counts all calls.
type panicRepo struct {
	*mockRepository
	claimCalls  atomic.Int32
	expiryCalls atomic.Int32
}

func (p *panicRepo) ClaimDueReminders(ctx context.Context) ([]*ContractReminder, error) {
	if p.claimCalls.Add(1) == 1 {
		panic("boom in reminder tick")
	}
	return p.mockRepository.ClaimDueReminders(ctx)
}

func (p *panicRepo) ExpireContracts(ctx context.Context) (int64, error) {
	if p.expiryCalls.Add(1) == 1 {
		panic("boom in expiry tick")
	}
	return p.mockRepository.ExpireContracts(ctx)
}

// A panic in the first reminder tick and the first expiry tick must not end
// Run: both tickers fire at least twice.
func TestReminderWorker_Run_SurvivesPanicInTicks(t *testing.T) {
	repo := &panicRepo{mockRepository: newMockRepository()}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 4}))
	w := &ReminderWorker{
		repo:           repo,
		logger:         logger,
		pollInterval:   5 * time.Millisecond,
		expiryInterval: 5 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	deadline := time.After(2 * time.Second)
	for repo.claimCalls.Load() < 2 || repo.expiryCalls.Load() < 2 {
		select {
		case <-deadline:
			cancel()
			t.Fatalf("worker stopped after a panic: claim=%d expiry=%d", repo.claimCalls.Load(), repo.expiryCalls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run returned %v", err)
	}
}
