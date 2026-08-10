package caldav

// Covers the write/read paths of PushSubscriptionService that
// tenant_write_test.go does not touch (Subscribe only) plus the
// fire-and-forget HTTP delivery in push_notifier.go. Uses a real Postgres
// connection (SkipIfNoDB) because GetSubscriptionsForCollection's expiry
// filter and CleanupExpired's DELETE are SQL, not Go — a stub would not
// exercise either.

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/testutil"
)

func TestPushSubscriptionService_UnsubscribeAndGetForCollection(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Sub Test Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-sub-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)
	collID := uuid.New()

	subA, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, "https://push.example.com/a", "topic-a", 0)
	if err != nil {
		t.Fatalf("Subscribe A: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", subA.ID)

	subB, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, "https://push.example.com/b", "topic-b", 0)
	if err != nil {
		t.Fatalf("Subscribe B: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", subB.ID)

	subs, err := svc.GetSubscriptionsForCollection(ctx, "calendar", collID)
	if err != nil {
		t.Fatalf("GetSubscriptionsForCollection: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(subs))
	}

	// A push URL that doesn't match any row must be a no-op, not an error.
	if err := svc.Unsubscribe(ctx, userID, "calendar", collID, "https://push.example.com/does-not-exist"); err != nil {
		t.Fatalf("Unsubscribe (no match): %v", err)
	}
	subs, err = svc.GetSubscriptionsForCollection(ctx, "calendar", collID)
	if err != nil {
		t.Fatalf("GetSubscriptionsForCollection after no-op unsubscribe: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("no-op Unsubscribe removed a row: expected 2, got %d", len(subs))
	}

	if err := svc.Unsubscribe(ctx, userID, "calendar", collID, subA.PushURL); err != nil {
		t.Fatalf("Unsubscribe A: %v", err)
	}
	subs, err = svc.GetSubscriptionsForCollection(ctx, "calendar", collID)
	if err != nil {
		t.Fatalf("GetSubscriptionsForCollection after unsubscribe: %v", err)
	}
	if len(subs) != 1 || subs[0].ID != subB.ID {
		t.Fatalf("expected only subB to remain, got %+v", subs)
	}
}

func TestPushSubscriptionService_UnsubscribeByURLAndExpiryFiltering(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Sub Expiry Test Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-sub-exp-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)
	collID := uuid.New()

	active, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, "https://push.example.com/active", "", 0)
	if err != nil {
		t.Fatalf("Subscribe active: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", active.ID)

	expired, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, "https://push.example.com/expired", "", 0)
	if err != nil {
		t.Fatalf("Subscribe expired: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", expired.ID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`UPDATE caldav_push_subscriptions SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = $1`,
		expired.ID,
	); err != nil {
		t.Fatalf("backdate expired subscription: %v", err)
	}

	subs, err := svc.GetSubscriptionsForCollection(ctx, "calendar", collID)
	if err != nil {
		t.Fatalf("GetSubscriptionsForCollection: %v", err)
	}
	if len(subs) != 1 || subs[0].ID != active.ID {
		t.Fatalf("expected only the active subscription, got %+v", subs)
	}

	// UnsubscribeByURL matches on push_url alone (used for 410 Gone cleanup),
	// no user/collection scoping.
	if err := svc.UnsubscribeByURL(ctx, active.PushURL); err != nil {
		t.Fatalf("UnsubscribeByURL: %v", err)
	}
	testutil.AssertRowCount(t, pool, sysCtx, "caldav_push_subscriptions", active.ID, 0)
}

func TestPushSubscriptionService_CleanupExpired(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Sub Cleanup Test Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-sub-cleanup-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)

	active, err := svc.Subscribe(ctx, userID, tenant, "calendar", uuid.New(), "https://push.example.com/cleanup-active", "", 0)
	if err != nil {
		t.Fatalf("Subscribe active: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", active.ID)

	expired, err := svc.Subscribe(ctx, userID, tenant, "calendar", uuid.New(), "https://push.example.com/cleanup-expired", "", 0)
	if err != nil {
		t.Fatalf("Subscribe expired: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", expired.ID)

	sysCtx := testutil.WithSystemCtx(context.Background())
	if _, err := pool.Exec(sysCtx,
		`UPDATE caldav_push_subscriptions SET expires_at = NOW() - INTERVAL '1 hour' WHERE id = $1`,
		expired.ID,
	); err != nil {
		t.Fatalf("backdate expired subscription: %v", err)
	}

	// CleanupExpired has no tenant scoping (it runs tenant-wide), so other
	// concurrently-expired rows may also be swept up — only assert on our
	// own two rows, not on the exact returned count.
	if _, err := svc.CleanupExpired(sysCtx); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	testutil.AssertRowCount(t, pool, sysCtx, "caldav_push_subscriptions", expired.ID, 0)
	testutil.AssertRowCount(t, pool, sysCtx, "caldav_push_subscriptions", active.ID, 1)
}

func TestPushNotifier_NotifyCollectionChanged_DeliversPayloadAndSkipsFailingSubscriber(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Notifier Test Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-notify-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	received := make(chan pushMessage, 1)
	var gotContentType string
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		var msg pushMessage
		if err := xml.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("decode push message: %v", err)
		}
		received <- msg
		w.WriteHeader(http.StatusOK)
	}))
	defer good.Close()

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)
	collID := uuid.New()

	// Nothing listens on 127.0.0.1:1 — the connection is refused immediately,
	// simulating a dead subscriber without a real timeout wait.
	failing, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, "http://127.0.0.1:1/push", "", 0)
	if err != nil {
		t.Fatalf("Subscribe failing: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", failing.ID)

	goodSub, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, good.URL, "my-topic", 0)
	if err != nil {
		t.Fatalf("Subscribe good: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", goodSub.ID)

	notifier := NewPushNotifier(svc)
	if err := notifier.NotifyCollectionChanged(ctx, "calendar", collID); err != nil {
		t.Fatalf("NotifyCollectionChanged: %v", err)
	}

	select {
	case msg := <-received:
		if msg.Topic != "my-topic" {
			t.Errorf("expected topic %q, got %q", "my-topic", msg.Topic)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("good subscriber never received a push notification — the failing subscriber may have blocked delivery")
	}

	if gotContentType != "application/xml; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "application/xml; charset=utf-8", gotContentType)
	}
}

// TestPushNotifier_NotifyCollectionChanged_410Gone_DocumentsMissingContextOnRemoval
// documents a real gap found while writing this coverage: sendNotification's
// 410-Gone branch (push_notifier.go) calls UnsubscribeByURL with
// `context.WithTimeout(context.Background(), 5*time.Second)` — bare, with
// neither a tenant nor a system-context marker. Per
// internal/database.TestPrepareConn_NoTenantNoSystem, that combination is the
// intentional "safe default for accidentally-unwrapped paths": RLS admits
// nothing, so the DELETE silently affects zero rows. In production this means
// a subscription whose endpoint returns 410 Gone is NEVER actually removed —
// every future NotifyCollectionChanged keeps POSTing to a dead URL forever.
// Not fixed inline (real behavior change, not a coverage change) — logged in
// JOURNAL.md as a new fix-* backlog unit. This test pins the current (broken)
// behavior so a future fix flips it red, proving the fix.
func TestPushNotifier_NotifyCollectionChanged_410Gone_DocumentsMissingContextOnRemoval(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenant := uuid.New()
	testutil.EnsureTenant(t, pool, tenant, "Push Notifier Gone Test Tenant")
	userID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenant,
		"email":         fmt.Sprintf("push-notify-gone-%s@tenanta.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", userID)

	handled := make(chan struct{}, 1)
	gone := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		handled <- struct{}{}
	}))
	defer gone.Close()

	ctx := testutil.WithTenantCtx(context.Background(), tenant)
	svc := NewPushSubscriptionService(pool)
	collID := uuid.New()

	sub, err := svc.Subscribe(ctx, userID, tenant, "calendar", collID, gone.URL, "", 0)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "caldav_push_subscriptions", sub.ID)

	notifier := NewPushNotifier(svc)
	if err := notifier.NotifyCollectionChanged(ctx, "calendar", collID); err != nil {
		t.Fatalf("NotifyCollectionChanged: %v", err)
	}

	select {
	case <-handled:
	case <-time.After(3 * time.Second):
		t.Fatal("410 subscriber never received the push notification")
	}

	// Give the async removal goroutine time to run (it races the assertion
	// below, but never wins it — that's the gap).
	time.Sleep(500 * time.Millisecond)

	sysCtx := testutil.WithSystemCtx(context.Background())
	testutil.AssertRowCount(t, pool, sysCtx, "caldav_push_subscriptions", sub.ID, 1)
}
