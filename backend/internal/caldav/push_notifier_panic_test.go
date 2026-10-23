package caldav

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

// panicOnceTransport panics on the first request and counts every request.
type panicOnceTransport struct{ calls atomic.Int32 }

func (p *panicOnceTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if p.calls.Add(1) == 1 {
		panic("boom in push transport")
	}
	return nil, fmt.Errorf("refused")
}

// A panic while pushing to one subscriber must neither crash the process nor
// skip the remaining subscribers of the same collection.
func TestPushNotifier_NotifyCollectionChanged_PanicForOneSubscriberDoesNotSkipTheRest(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Notifier Panic Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-panic-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)
	collID := uuid.New()
	for _, u := range []string{"https://push-a.example.test/p", "https://push-b.example.test/p"} {
		sub, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, u, "", 0)
		if err != nil {
			t.Fatalf("Subscribe %s: %v", u, err)
		}
		defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", sub.ID)
	}

	tr := &panicOnceTransport{}
	notifier := newPushNotifier(svc)
	notifier.httpClient = &http.Client{Transport: tr, Timeout: pushNotifyTimeout}
	if err := notifier.NotifyCollectionChanged(ctx, "calendar", collID); err != nil {
		t.Fatalf("NotifyCollectionChanged: %v", err)
	}

	deadline := time.After(3 * time.Second)
	for tr.calls.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("second subscriber never attempted after the panic: %d request(s)", tr.calls.Load())
		case <-time.After(5 * time.Millisecond):
		}
	}
}
