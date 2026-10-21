package lexware

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// panicEmitter panics on every Emit and counts the calls; SyncContactsWithConfig
// emits first, so each scheduled contact tick panics once.
type panicEmitter struct{ calls atomic.Int32 }

func (e *panicEmitter) Emit(context.Context, EventPayload) error {
	e.calls.Add(1)
	panic("emitter boom")
}

// A panic inside one tick must not end the loop: the next tick still runs.
// Asserting a second call proves it -- a test with a single call would also pass
// if the loop died after the first panic.
func TestRunTenantScheduler_SurvivesPanicInTick(t *testing.T) {
	em := &panicEmitter{}
	svc := newTestService(nil, nil, nil)
	svc.emitter = em
	s := NewScheduler(svc, nil, nil)
	s.intervalUnit = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	ts := &tenantScheduler{cancel: cancel, configID: uuid.New(), tenantID: uuid.New()}
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runTenantScheduler(ctx, ts, &models.LexwareSyncConfig{
			ContactSyncEnabled:     true,
			ContactSyncIntervalMin: 1,
		})
	}()

	deadline := time.After(5 * time.Second)
	for em.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("tick ran %d time(s), want >= 2: loop died after the panic", em.calls.Load())
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runTenantScheduler did not return after ctx cancellation")
	}
}
